package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ds "github.com/sealdice/dicescript"
	"gopkg.in/yaml.v3"
)

// GameSystemTemplateV2 新版本的游戏系统模板结构
type GameSystemTemplateV2 struct {
	Name        string   `yaml:"name"`        // 模板名字
	FullName    string   `yaml:"fullName"`    // 全名
	Authors     []string `yaml:"authors"`     // 作者
	Version     string   `yaml:"version"`     // 版本
	UpdatedTime string   `yaml:"updatedTime"` // 更新日期
	TemplateVer string   `yaml:"templateVer"` // 模板版本
	PreloadCode string   `yaml:"preloadCode"` // 预加载代码
	Attrs       Attrs    `yaml:"attrs"`       // 属性配置
	Alias       Alias    `yaml:"alias"`       // 别名配置
	Commands    Commands `yaml:"commands"`    // 命令配置

	AliasMap *SyncMap[string, string] `json:"-" yaml:"-"` // 别名/同义词
}

// Attrs 属性配置
type Attrs struct {
	Defaults         map[string]int    `yaml:"defaults"`         // 默认值
	DefaultsComputed map[string]string `yaml:"defaultsComputed"` // 计算默认值
	DetailOverwrite  map[string]string `yaml:"detailOverwrite"`  // 详情重写
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
	DiceSides     int      `yaml:"diceSides"`     // 骰子面数
	DiceSidesExpr string   `yaml:"diceSidesExpr"` // 骰子面数表达式
	EnableTip     string   `yaml:"enableTip"`     // 启用提示
	Keys          []string `yaml:"keys"`          // 可用于 .set xxx 的key
	RelatedExt    []string `yaml:"relatedExt"`    // 关联扩展
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
	Top         []string          `yaml:"top"`         // 锁定在前面的几个
	SortBy      string            `yaml:"sortBy"`      // 排序方式
	Ignores     []string          `yaml:"ignores"`     // 忽略的属性
	ShowValueAs map[string]string `yaml:"showValueAs"` // 影响单个属性的右侧内容
	ShowKeyAs   map[string]string `yaml:"showKeyAs"`   // 影响单个属性的左侧内容
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
	t.AliasMap = new(SyncMap[string, string])
	alias := t.Alias
	for k, v := range alias {
		for _, i := range v {
			t.AliasMap.Store(strings.ToLower(i), k)
		}
		t.AliasMap.Store(strings.ToLower(k), k)
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
func (t *GameSystemTemplateV2) GetDefaultValue(varname string) (*ds.VMValue, string, bool, bool) {
	var detail string

	// 需要优化成提前编译computed，这样直接载入就行
	// if expr, exists := t.Attrs.DefaultsComputed[varname]; exists {
	// 	return ds.NewIntVal(0), "", true, true
	// }

	// 具体语句
	if val, exists := t.Attrs.Defaults[varname]; exists {
		return ds.NewIntVal(ds.IntType(val)), detail, false, true
	}

	return nil, detail, false, false

}

// GetDefaultValueEx0 获取默认值 四个返回值 val, detail, computed, exists
func (t *GameSystemTemplateV2) GetDefaultValueEx0(ctx *GameState, varname string) (*ds.VMValue, string, bool, bool) {
	name := t.GetAlias(varname)
	var detail string

	// 先计算computed
	if expr, exists := t.Attrs.DefaultsComputed[name]; exists {
		ctx.SystemTemplate = t
		// 也许有点依赖这个东西？有更好的方式吗？可以更加全局的去加载吗（比如和vm创建伴生加载）？
		ctx.VM.Run(t.PreloadCode)
		ctx.VM.Run(expr)

		// if r.vm.Error == nil {
		// 	detail = r.vm.GetDetailText()
		// 	return &r.VMValue, detail, r.vm.IsCalculateExists() || r.vm.IsComputedLoaded, true
		// } else {
		// 	return ds.NewStrVal("代码:" + expr + "\n" + "报错:" + r.vm.Error.Error()), "", true, true
		// }
	}

	// if val, exists := t.Attrs.Defaults[name]; exists {
	// 	return ds.NewIntVal(ds.IntType(val)), detail, false, true
	// }

	// TODO: 以空值填充，这是vm v1的行为，未来需要再次评估其合理性
	return ds.NewIntVal(0), detail, false, false
}
