package converter

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/sealdice/smallseal/dice/types"
)

type NameTemplateItemV1 struct {
	Template string `json:"template" yaml:"template"`
	HelpText string `json:"helpText" yaml:"helpText"`
}

type AttrConfigV1 struct {
	Top          []string          `json:"top" yaml:"top,flow"`
	SortBy       string            `json:"sortBy" yaml:"sortBy"`
	Ignores      []string          `json:"ignores" yaml:"ignores"`
	ShowAs       map[string]string `json:"showAs" yaml:"showAs"`
	ShowAsKey    map[string]string `json:"showAsKey" yaml:"showAsKey"`
	ItemsPerLine int               `json:"itemsPerLine" yaml:"itemsPerLine"`
}

type SetConfigV1 struct {
	RelatedExt    []string `json:"relatedExt" yaml:"relatedExt"`
	DiceSides     int      `json:"diceSides" yaml:"diceSides"`
	DiceSidesExpr string   `json:"diceSidesExpr" yaml:"diceSidesExpr"`
	Keys          []string `json:"keys" yaml:"keys"`
	EnableTip     string   `json:"enableTip" yaml:"enableTip"`
}

type TemplateV1 struct {
	Name             string                        `json:"name" yaml:"name"`
	FullName         string                        `json:"fullName" yaml:"fullName"`
	Authors          []string                      `json:"authors" yaml:"authors"`
	Version          string                        `json:"version" yaml:"version"`
	UpdatedTime      string                        `json:"updatedTime" yaml:"updatedTime"`
	TemplateVer      string                        `json:"templateVer" yaml:"templateVer"`
	NameTemplate     map[string]NameTemplateItemV1 `json:"nameTemplate" yaml:"nameTemplate"`
	AttrConfig       AttrConfigV1                  `json:"attrConfig" yaml:"attrConfig"`
	SetConfig        SetConfigV1                   `json:"setConfig" yaml:"setConfig"`
	Defaults         map[string]int                `json:"defaults" yaml:"defaults"`
	DefaultsComputed map[string]string             `json:"defaultsComputed" yaml:"defaultsComputed"`
	DetailOverwrite  map[string]string             `json:"detailOverwrite" yaml:"detailOverwrite"`
	Alias            map[string][]string           `json:"alias" yaml:"alias"`
	TextMap          map[string]any                `json:"textMap" yaml:"textMap"`
	TextMapHelpInfo  map[string]any                `json:"textMapHelpInfo" yaml:"TextMapHelpInfo"`
	PreloadCode      string                        `json:"preloadCode" yaml:"preloadCode"`
}

type Output struct {
	*types.GameSystemTemplateV2 `yaml:",inline"`
	TextMap                     map[string]any `yaml:"textMap,omitempty"`
	TextMapHelpInfo             map[string]any `yaml:"textMapHelpInfo,omitempty"`
}

var detailFuncExpr = regexp.MustCompile("^[\\p{L}_][\\p{L}\\p{N}_]*\\s*\\(.*\\)$")

// ParseTemplate unmarshals the legacy template according to the desired format ("json" or "yaml").
func ParseTemplate(data []byte, format string) (*TemplateV1, error) {
	var tmpl TemplateV1
	var err error

	switch strings.ToLower(format) {
	case "json":
		err = json.Unmarshal(data, &tmpl)
	case "yaml", "yml":
		err = yaml.Unmarshal(data, &tmpl)
	default:
		err = fmt.Errorf("unsupported input format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse v1 template: %w", err)
	}

	return &tmpl, nil
}

// Convert projects a v1 template into the V2 data layout that SmallSeal expects.
func Convert(src *TemplateV1, overrideTemplateVer string) *Output {
	dst := &types.GameSystemTemplateV2{
		Name:        src.Name,
		FullName:    src.FullName,
		Authors:     cloneStrings(src.Authors),
		Version:     src.Version,
		UpdatedTime: src.UpdatedTime,
		TemplateVer: src.TemplateVer,
		InitScript:  src.PreloadCode,
		Attrs: types.Attrs{
			Defaults:         make(map[string]int, len(src.Defaults)),
			DefaultsComputed: make(map[string]string, len(src.DefaultsComputed)),
			DetailOverwrite:  make(map[string]string, len(src.DetailOverwrite)),
		},
		Alias: make(types.Alias, len(src.Alias)),
		Commands: types.Commands{
			Set: types.SetConfig{
				EnableTip:  src.SetConfig.EnableTip,
				Keys:       cloneStrings(src.SetConfig.Keys),
				RelatedExt: cloneStrings(src.SetConfig.RelatedExt),
			},
			Sn: make(types.SnConfig, len(src.NameTemplate)),
			St: types.StConfig{
				Show: types.StShowConfig{
					Top:          cloneStrings(src.AttrConfig.Top),
					SortBy:       src.AttrConfig.SortBy,
					Ignores:      cloneStrings(src.AttrConfig.Ignores),
					ShowValueAs:  cloneStringMap(src.AttrConfig.ShowAs),
					ShowKeyAs:    cloneStringMap(src.AttrConfig.ShowAsKey),
					ItemsPerLine: src.AttrConfig.ItemsPerLine,
				},
			},
		},
	}

	if overrideTemplateVer != "" {
		dst.TemplateVer = overrideTemplateVer
	} else if dst.TemplateVer == "" {
		dst.TemplateVer = "2.0"
	}

	for k, v := range src.Defaults {
		dst.Attrs.Defaults[k] = v
	}

	for k, v := range src.DefaultsComputed {
		dst.Attrs.DefaultsComputed[k] = v
	}

	for k, v := range src.DetailOverwrite {
		dst.Attrs.DetailOverwrite[k] = normalizeDetailOverwrite(v)
	}

	if len(dst.Attrs.Defaults) == 0 {
		dst.Attrs.Defaults = nil
	}
	if len(dst.Attrs.DefaultsComputed) == 0 {
		dst.Attrs.DefaultsComputed = nil
	}
	if len(dst.Attrs.DetailOverwrite) == 0 {
		dst.Attrs.DetailOverwrite = nil
	}

	for k, v := range src.Alias {
		dst.Alias[k] = cloneStrings(v)
	}
	if len(dst.Alias) == 0 {
		dst.Alias = nil
	}

	if expr := strings.TrimSpace(src.SetConfig.DiceSidesExpr); expr != "" {
		dst.Commands.Set.DiceSidesExpr = expr
	} else if src.SetConfig.DiceSides != 0 {
		dst.Commands.Set.DiceSidesExpr = strconv.Itoa(src.SetConfig.DiceSides)
	}

	if len(dst.Commands.Set.Keys) == 0 {
		dst.Commands.Set.Keys = nil
	}
	if len(dst.Commands.Set.RelatedExt) == 0 {
		dst.Commands.Set.RelatedExt = nil
	}

	for key, tmpl := range src.NameTemplate {
		dst.Commands.Sn[key] = types.SnTemplate{
			Template: tmpl.Template,
			HelpText: tmpl.HelpText,
		}
	}
	if len(dst.Commands.Sn) == 0 {
		dst.Commands.Sn = nil
	}

	show := &dst.Commands.St.Show
	if len(show.Top) == 0 {
		show.Top = nil
	}
	if len(show.Ignores) == 0 {
		show.Ignores = nil
	}
	if len(show.ShowValueAs) == 0 {
		show.ShowValueAs = nil
	}
	if len(show.ShowKeyAs) == 0 {
		show.ShowKeyAs = nil
	}

	return &Output{
		GameSystemTemplateV2: dst,
		TextMap:              src.TextMap,
		TextMapHelpInfo:      src.TextMapHelpInfo,
	}
}

// MarshalOutput renders the converted template in the requested format.
func MarshalOutput(template *Output, format string) ([]byte, error) {
	switch strings.ToLower(format) {
	case "json":
		return json.MarshalIndent(template, "", "  ")
	case "yaml", "yml":
		return yaml.Marshal(template)
	default:
		return nil, fmt.Errorf("unsupported output format: %s", format)
	}
}

func normalizeDetailOverwrite(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
		return trimmed
	}
	if detailFuncExpr.MatchString(trimmed) {
		return "{" + trimmed + "}"
	}
	return trimmed
}

func cloneStrings(src []string) []string {
	if len(src) == 0 {
		return nil
	}
	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
