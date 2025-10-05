package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sealdice/smallseal/dice/attrs"
	"github.com/tidwall/buntdb"
)

var (
	keyEncoding     = base64.RawURLEncoding
	errAttrsMissing = errors.New("attributes item not found")
)

type storedAttrRecord struct {
	ID               string
	Data             string
	Name             string
	SheetType        string
	OwnerID          string
	AttrsType        string
	IsHidden         bool
	LastModifiedTime int64
	LastUsedTime     int64
}

type buntAttrsIO struct {
	db *buntdb.DB
}

func newBuntAttrsIO(db *buntdb.DB) *buntAttrsIO {
	return &buntAttrsIO{db: db}
}

func encodeKeyPart(value string) string {
	if value == "" {
		return "_"
	}
	return keyEncoding.EncodeToString([]byte(value))
}

func decodeKeyPart(value string) (string, error) {
	if value == "_" {
		return "", nil
	}
	decoded, err := keyEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func attrKey(id string) string {
	return "attr:" + encodeKeyPart(id)
}

func ownerIndexKey(ownerID, attrID string) string {
	return fmt.Sprintf("owner:%s:%s", encodeKeyPart(ownerID), encodeKeyPart(attrID))
}

func ownerIndexPrefix(ownerID string) string {
	return fmt.Sprintf("owner:%s:", encodeKeyPart(ownerID))
}

func nameIndexKey(ownerID, name string) string {
	return fmt.Sprintf("name:%s:%s", encodeKeyPart(ownerID), encodeKeyPart(name))
}

func bindKey(groupID, userID string) string {
	return fmt.Sprintf("bind:%s:%s", encodeKeyPart(groupID), encodeKeyPart(userID))
}

func bindRevKey(attrID, groupID, userID string) string {
	return fmt.Sprintf("bindrev:%s:%s:%s", encodeKeyPart(attrID), encodeKeyPart(groupID), encodeKeyPart(userID))
}

func bindRevPrefix(attrID string) string {
	return fmt.Sprintf("bindrev:%s:", encodeKeyPart(attrID))
}

func (io *buntAttrsIO) loadAttr(tx *buntdb.Tx, id string) (*storedAttrRecord, bool, error) {
	value, err := tx.Get(attrKey(id))
	if err != nil {
		if errors.Is(err, buntdb.ErrNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	rec := &storedAttrRecord{}
	if err := json.Unmarshal([]byte(value), rec); err != nil {
		return nil, false, err
	}
	if rec.ID == "" {
		rec.ID = id
	}
	return rec, true, nil
}

func (io *buntAttrsIO) persistAttr(tx *buntdb.Tx, rec *storedAttrRecord, prevOwner, prevName string) error {
	if rec == nil {
		return nil
	}
	payload, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	if _, _, err := tx.Set(attrKey(rec.ID), string(payload), nil); err != nil {
		return err
	}
	if prevOwner != "" {
		if prevOwner != rec.OwnerID {
			if _, err := tx.Delete(ownerIndexKey(prevOwner, rec.ID)); err != nil && !errors.Is(err, buntdb.ErrNotFound) {
				return err
			}
		}
		if prevName != "" && (prevOwner != rec.OwnerID || prevName != rec.Name) {
			if _, err := tx.Delete(nameIndexKey(prevOwner, prevName)); err != nil && !errors.Is(err, buntdb.ErrNotFound) {
				return err
			}
		}
	}
	if rec.OwnerID != "" {
		if _, _, err := tx.Set(ownerIndexKey(rec.OwnerID, rec.ID), rec.ID, nil); err != nil {
			return err
		}
		if rec.Name != "" {
			if _, _, err := tx.Set(nameIndexKey(rec.OwnerID, rec.Name), rec.ID, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func (rec *storedAttrRecord) toItem() *attrs.AttributesItem {
	item := &attrs.AttributesItem{
		ID:               rec.ID,
		Data:             []byte{},
		Name:             rec.Name,
		SheetType:        rec.SheetType,
		OwnerId:          rec.OwnerID,
		AttrsType:        rec.AttrsType,
		IsHidden:         rec.IsHidden,
		LastModifiedTime: rec.LastModifiedTime,
		LastUsedTime:     rec.LastUsedTime,
		IsSaved:          true,
	}
	if rec.Data != "" {
		if decoded, err := base64.StdEncoding.DecodeString(rec.Data); err == nil {
			item.Data = decoded
		}
	}
	return item
}

func (io *buntAttrsIO) GetById(id string) (*attrs.AttributesItem, error) {
	if id == "" {
		return nil, errAttrsMissing
	}
	var result *attrs.AttributesItem
	err := io.db.Update(func(tx *buntdb.Tx) error {
		rec, exists, err := io.loadAttr(tx, id)
		if err != nil {
			return err
		}
		if !exists {
			return errAttrsMissing
		}
		prevOwner, prevName := rec.OwnerID, rec.Name
		rec.LastUsedTime = time.Now().Unix()
		if err := io.persistAttr(tx, rec, prevOwner, prevName); err != nil {
			return err
		}
		result = rec.toItem()
		return nil
	})
	if errors.Is(err, errAttrsMissing) {
		return nil, err
	}
	return result, err
}

func (io *buntAttrsIO) Puts(items []*attrs.AttrsUpsertParams) error {
	if len(items) == 0 {
		return nil
	}
	now := time.Now().Unix()
	return io.db.Update(func(tx *buntdb.Tx) error {
		for _, param := range items {
			if param == nil {
				continue
			}
			if param.Id == "" {
				return errors.New("id cannot be empty")
			}
			rec, exists, err := io.loadAttr(tx, param.Id)
			if err != nil {
				return err
			}
			prevOwner, prevName := "", ""
			if !exists {
				rec = &storedAttrRecord{ID: param.Id}
			} else {
				prevOwner, prevName = rec.OwnerID, rec.Name
			}
			rec.Name = param.Name
			rec.SheetType = param.SheetType
			rec.OwnerID = param.OwnerId
			rec.AttrsType = param.AttrsType
			rec.IsHidden = param.IsHidden
			rec.LastModifiedTime = now
			if rec.LastUsedTime == 0 {
				rec.LastUsedTime = now
			}
			rec.Data = base64.StdEncoding.EncodeToString(param.Data)
			if err := io.persistAttr(tx, rec, prevOwner, prevName); err != nil {
				return err
			}
		}
		return nil
	})
}

func (io *buntAttrsIO) DeleteById(id string) error {
	if id == "" {
		return nil
	}
	return io.db.Update(func(tx *buntdb.Tx) error {
		rec, exists, err := io.loadAttr(tx, id)
		if err != nil {
			return err
		}
		if !exists {
			return errAttrsMissing
		}
		if rec.OwnerID != "" {
			if _, err := tx.Delete(ownerIndexKey(rec.OwnerID, rec.ID)); err != nil && !errors.Is(err, buntdb.ErrNotFound) {
				return err
			}
			if rec.Name != "" {
				if _, err := tx.Delete(nameIndexKey(rec.OwnerID, rec.Name)); err != nil && !errors.Is(err, buntdb.ErrNotFound) {
					return err
				}
			}
		}
		if _, err := tx.Delete(attrKey(id)); err != nil && !errors.Is(err, buntdb.ErrNotFound) {
			return err
		}
		_, err = io.removeAllBindingsTx(tx, id)
		return err
	})
}

func (io *buntAttrsIO) ListByUid(userID string) ([]*attrs.AttributesItem, error) {
	if userID == "" {
		return []*attrs.AttributesItem{}, nil
	}
	items := []*attrs.AttributesItem{}
	err := io.db.View(func(tx *buntdb.Tx) error {
		pattern := ownerIndexPrefix(userID) + "*"
		var innerErr error
		err := tx.AscendKeys(pattern, func(key, value string) bool {
			if innerErr != nil {
				return false
			}
			rec, exists, err := io.loadAttr(tx, value)
			if err != nil {
				innerErr = err
				return false
			}
			if !exists {
				return true
			}
			items = append(items, rec.toItem())
			return true
		})
		if innerErr != nil {
			return innerErr
		}
		return err
	})
	return items, err
}

func (io *buntAttrsIO) GetByUidAndName(userID string, name string) (*attrs.AttributesItem, error) {
	if userID == "" || name == "" {
		return nil, nil
	}
	var result *attrs.AttributesItem
	err := io.db.Update(func(tx *buntdb.Tx) error {
		attrID, err := tx.Get(nameIndexKey(userID, name))
		if err != nil {
			if errors.Is(err, buntdb.ErrNotFound) {
				return nil
			}
			return err
		}
		rec, exists, err := io.loadAttr(tx, attrID)
		if err != nil {
			return err
		}
		if !exists {
			if _, err := tx.Delete(nameIndexKey(userID, name)); err != nil && !errors.Is(err, buntdb.ErrNotFound) {
				return err
			}
			return nil
		}
		prevOwner, prevName := rec.OwnerID, rec.Name
		rec.LastUsedTime = time.Now().Unix()
		if err := io.persistAttr(tx, rec, prevOwner, prevName); err != nil {
			return err
		}
		result = rec.toItem()
		return nil
	})
	return result, err
}

func (io *buntAttrsIO) BindingIdGet(groupID string, userID string) (string, error) {
	var result string
	err := io.db.View(func(tx *buntdb.Tx) error {
		value, err := tx.Get(bindKey(groupID, userID))
		if err != nil {
			if errors.Is(err, buntdb.ErrNotFound) {
				result = ""
				return nil
			}
			return err
		}
		result = value
		return nil
	})
	return result, err
}

func (io *buntAttrsIO) Bind(groupID string, userID string, attrsID string) error {
	if attrsID == "" {
		return errors.New("attrs not found")
	}
	return io.db.Update(func(tx *buntdb.Tx) error {
		if _, exists, err := io.loadAttr(tx, attrsID); err != nil {
			return err
		} else if !exists {
			return errAttrsMissing
		}
		key := bindKey(groupID, userID)
		prevID, err := tx.Get(key)
		if err != nil && !errors.Is(err, buntdb.ErrNotFound) {
			return err
		}
		if _, _, err := tx.Set(key, attrsID, nil); err != nil {
			return err
		}
		if prevID != "" && prevID != attrsID {
			if _, err := tx.Delete(bindRevKey(prevID, groupID, userID)); err != nil && !errors.Is(err, buntdb.ErrNotFound) {
				return err
			}
		}
		if _, _, err := tx.Set(bindRevKey(attrsID, groupID, userID), "", nil); err != nil {
			return err
		}
		return nil
	})
}

func (io *buntAttrsIO) Unbind(groupID string, userID string) error {
	return io.db.Update(func(tx *buntdb.Tx) error {
		key := bindKey(groupID, userID)
		attrID, err := tx.Get(key)
		if err != nil {
			if errors.Is(err, buntdb.ErrNotFound) {
				return nil
			}
			return err
		}
		if _, err := tx.Delete(key); err != nil && !errors.Is(err, buntdb.ErrNotFound) {
			return err
		}
		if _, err := tx.Delete(bindRevKey(attrID, groupID, userID)); err != nil && !errors.Is(err, buntdb.ErrNotFound) {
			return err
		}
		return nil
	})
}

func (io *buntAttrsIO) UnbindAll(attrsID string) (int64, error) {
	var removed int64
	err := io.db.Update(func(tx *buntdb.Tx) error {
		count, err := io.removeAllBindingsTx(tx, attrsID)
		if err != nil {
			return err
		}
		removed = count
		return nil
	})
	return removed, err
}

func (io *buntAttrsIO) removeAllBindingsTx(tx *buntdb.Tx, attrsID string) (int64, error) {
	prefix := bindRevPrefix(attrsID)
	pairs := make([][2]string, 0)
	var innerErr error
	err := tx.AscendKeys(prefix+"*", func(key, value string) bool {
		if innerErr != nil {
			return false
		}
		parts := strings.Split(key, ":")
		if len(parts) != 4 {
			innerErr = fmt.Errorf("invalid binding key: %s", key)
			return false
		}
		groupID, err := decodeKeyPart(parts[2])
		if err != nil {
			innerErr = err
			return false
		}
		userID, err := decodeKeyPart(parts[3])
		if err != nil {
			innerErr = err
			return false
		}
		pairs = append(pairs, [2]string{groupID, userID})
		return true
	})
	if innerErr != nil {
		return 0, innerErr
	}
	if err != nil {
		return 0, err
	}
	var removed int64
	for _, pair := range pairs {
		key := bindKey(pair[0], pair[1])
		value, err := tx.Get(key)
		if err != nil && !errors.Is(err, buntdb.ErrNotFound) {
			return removed, err
		}
		if value == attrsID {
			if _, err := tx.Delete(key); err != nil && !errors.Is(err, buntdb.ErrNotFound) {
				return removed, err
			}
			removed++
		}
		if _, err := tx.Delete(bindRevKey(attrsID, pair[0], pair[1])); err != nil && !errors.Is(err, buntdb.ErrNotFound) {
			return removed, err
		}
	}
	return removed, nil
}

func (io *buntAttrsIO) BindingGroupIdList(attrsID string) ([]string, error) {
	groups := []string{}
	err := io.db.View(func(tx *buntdb.Tx) error {
		pattern := bindRevPrefix(attrsID) + "*"
		var innerErr error
		seen := map[string]struct{}{}
		err := tx.AscendKeys(pattern, func(key, value string) bool {
			if innerErr != nil {
				return false
			}
			parts := strings.Split(key, ":")
			if len(parts) != 4 {
				innerErr = fmt.Errorf("invalid binding key: %s", key)
				return false
			}
			groupID, err := decodeKeyPart(parts[2])
			if err != nil {
				innerErr = err
				return false
			}
			if _, ok := seen[groupID]; !ok {
				seen[groupID] = struct{}{}
				groups = append(groups, groupID)
			}
			return true
		})
		if innerErr != nil {
			return innerErr
		}
		return err
	})
	return groups, err
}
