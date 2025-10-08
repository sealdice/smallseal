package types

// 放这里不太合适，不过必要时还能继续拆

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/samber/lo"
	ds "github.com/sealdice/dicescript"
	"gopkg.in/yaml.v3"

	"github.com/sealdice/smallseal/utils"
)

// GameSystemTemplateV2 新版本的游戏系统模板结构
type GameSystemTemplateV2 struct {
	Name        string   `yaml:"name"`        // 模板名字
	FullName    string   `yaml:"fullName"`    // 全名
	Authors     []string `yaml:"authors"`     // 作者
	Version     string   `yaml:"version"`     // 版本
	UpdatedTime string   `yaml:"updatedTime"` // 更新日期
	TemplateVer string   `yaml:"templateVer"` // 模板版本
	InitScript  string   `yaml:"initScript"`  // 预加载代码
	Attrs       Attrs    `yaml:"attrs"`       // 属性配置
	Alias       Alias    `yaml:"alias"`       // 别名配置
	Commands    Commands `yaml:"commands"`    // 命令配置

	AliasMap          *utils.SyncMap[string, string]                                                                                                            `json:"-" yaml:"-"` // 别名/同义词
	HookValueLoadPost func(ctx *ds.Context, name string, curVal *ds.VMValue, doCompute func(curVal *ds.VMValue) *ds.VMValue, detail *ds.BufferSpan) *ds.VMValue `json:"-" yaml:"-"`
}

// Attrs 属性配置
type Attrs struct {
	Defaults         map[string]int    `yaml:"defaults"`         // 默认值
	DefaultsComputed map[string]string `yaml:"defaultsComputed"` // 计算默认值
	DetailOverwrite  map[string]string `yaml:"detailOverwrite"`  // 计算过程重写

	DefaultsComputedReal map[string]*ds.VMValue `json:"-" yaml:"-"`
}

// Alias 别名配置
type Alias map[string][]string

// Commands 命令配置
type Commands struct {
	Set SetConfig `yaml:"set"` // set命令配置
	Sn  SnConfig  `yaml:"sn"`  // sn命令配置
	St  StConfig  `yaml:"st"`  // st命令配置
}

// SetConfig set命令配置
type SetConfig struct {
	DiceSidesExpr string   `yaml:"diceSides"`  // 骰子面数表达式
	EnableTip     string   `yaml:"enableTip"`  // 启用提示
	Keys          []string `yaml:"keys"`       // 可用于 .set xxx 的key
	RelatedExt    []string `yaml:"relatedExt"` // 关联扩展
}

// SnConfig sn命令配置
type SnConfig map[string]SnTemplate

// SnTemplate sn模板配置
type SnTemplate struct {
	Template string `yaml:"template"` // 模板
	HelpText string `yaml:"helpText"` // 帮助文本
}

// StConfig st命令配置
type StConfig struct {
	Show StShowConfig `yaml:"show"` // show配置
}

// StShowConfig st show配置
type StShowConfig struct {
	Top          []string          `yaml:"top"`          // 锁定在前面的几个
	SortBy       string            `yaml:"sortBy"`       // 排序方式
	Ignores      []string          `yaml:"ignores"`      // 忽略的属性
	ShowValueAs  map[string]string `yaml:"showValueAs"`  // 影响单个属性的右侧内容
	ShowKeyAs    map[string]string `yaml:"showKeyAs"`    // 影响单个属性的左侧内容
	ItemsPerLine int               `yaml:"itemsPerLine"` // 每行显示多少个属性
}

// LoadGameSystemTemplate 从YAML或JSON文件加载游戏系统模板
func LoadGameSystemTemplate(filename string) (*GameSystemTemplateV2, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var template GameSystemTemplateV2
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &template)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
		}
	case ".json":
		err = json.Unmarshal(data, &template)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported file format: %s", ext)
	}

	template.Init()
	return &template, nil
}

// SaveGameSystemTemplate 将游戏系统模板保存到YAML或JSON文件
func SaveGameSystemTemplate(template *GameSystemTemplateV2, filename string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	var data []byte
	var err error

	switch ext {
	case ".yaml", ".yml":
		data, err = yaml.Marshal(template)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}
	case ".json":
		data, err = json.MarshalIndent(template, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
	default:
		return fmt.Errorf("unsupported file format: %s", ext)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (t *GameSystemTemplateV2) Init() {
	t.AliasMap = new(utils.SyncMap[string, string])
	alias := t.Alias
	for k, v := range alias {
		for _, i := range v {
			t.AliasMap.Store(strings.ToLower(i), k)
		}
		t.AliasMap.Store(strings.ToLower(k), k)
	}

	// 处理 DefaultsComputedReal
	t.Attrs.DefaultsComputedReal = make(map[string]*ds.VMValue)
	if t.Attrs.DefaultsComputed != nil {
		for k, v := range t.Attrs.DefaultsComputed {
			t.Attrs.DefaultsComputedReal[k] = ds.NewComputedVal(v)
		}
	}
}

// "github.com/go-creed/sat"
// var chsS2T = sat.DefaultDict()

func (t *GameSystemTemplateV2) GetAlias(varname string) string {
	k := strings.ToLower(varname)
	if t == nil {
		return varname
	}
	v2, exists := t.AliasMap.Load(k)
	if exists {
		varname = v2
	} else {
		// k = chsS2T.Read(k)
		v2, exists = t.AliasMap.Load(k)
		if exists {
			varname = v2
		}
	}

	return varname
}

// 默认值获取搞得太复杂了
// 这个是不经过计算的获取默认值，如果默认值是computed，那么返回的是computed
// 四个返回值 val, detail, computed, exists
func (t *GameSystemTemplateV2) GetDefaultValue(varname string) (*ds.VMValue, string, bool, bool) {
	var detail string

	// 需要优化成提前编译computed，这样直接载入就行
	if computed, exists := t.Attrs.DefaultsComputedReal[varname]; exists {
		return computed, "", true, true
	}

	// 具体语句
	if val, exists := t.Attrs.Defaults[varname]; exists {
		return ds.NewIntVal(ds.IntType(val)), detail, false, true
	}

	return nil, detail, false, false
}

func (t *GameSystemTemplateV2) GetShowKeyAs(ctx *MsgContext, k string) (string, error) {
	var r *ds.VMValue
	var err error
	// 如果存在showKeyAs，那么就根据表达式求值
	if expr, exists := t.Commands.St.Show.ShowKeyAs[k]; exists {
		r, _, err = ctx.EvalFString(expr, &ds.RollConfig{
			DefaultDiceSideExpr: ctx.GetDefaultDicePoints(),
		})
		if err == nil {
			k = r.ToString()
		}
	}
	return k, err
}

func (t *GameSystemTemplateV2) GetShowValueAs(ctx *MsgContext, k string) (*ds.VMValue, error) {
	var r *ds.VMValue
	var err error

	if expr, exists := t.Commands.St.Show.ShowValueAs[k]; exists {
		r, _, err = ctx.EvalFString(expr, &ds.RollConfig{
			DefaultDiceSideExpr: ctx.GetDefaultDicePoints(),
		})
		if err == nil {
			return r, nil
		}
	}

	// 通配值
	if expr, exists := t.Commands.St.Show.ShowValueAs["*"]; exists {
		// 这里存入k是因为接下来模板可能会用到原始key，例如 loadRaw(name)
		ctx.vm.StoreNameLocal("name", ds.NewStrVal(k))
		r, _, err = ctx.EvalFString(expr, nil)
		if err == nil {
			return r, nil
		}
		return ds.NewIntVal(0), err
	}

	// 没有二次覆盖，读取真实值
	return t.GetRealValue(ctx, k)
}

func (t *GameSystemTemplateV2) GetRealValueBase(ctx *MsgContext, k string) (*ds.VMValue, error) {
	// 获取实际值
	am := ctx.AttrsManager
	curAttrs := lo.Must(am.Load(ctx.Group.GroupId, ctx.Player.UserId))
	v, exists := curAttrs.Load(k)
	if exists {
		return v, nil
	}

	// 默认值
	v, _, _, exists = t.GetDefaultValue(k)
	if exists {
		return v, nil
	}

	// 不存在的值
	return nil, nil //nolint:nilnil
}

func (t *GameSystemTemplateV2) GetRealValue(ctx *MsgContext, k string) (*ds.VMValue, error) {
	v, err := t.GetRealValueBase(ctx, k)
	if v == nil && err == nil {
		// 不存在的值，强行补0
		return ds.NewIntVal(0), nil
	}
	return v, err
}
