package exts

import _ "embed"

// GameSystemTemplateAsset describes an embedded template bundle.
type GameSystemTemplateAsset struct {
	Filename string
	Data     []byte
}

var (
	//go:embed coc7.yaml
	builtinGameSystemTemplateCoc7 []byte
	//go:embed dnd5e.yaml
	builtinGameSystemTemplateDnd5e []byte
)

// BuiltinGameSystemTemplateAssets returns embedded template bundles.
func BuiltinGameSystemTemplateAssets() []GameSystemTemplateAsset {
	return []GameSystemTemplateAsset{
		{Filename: "coc7.yaml", Data: builtinGameSystemTemplateCoc7},
		{Filename: "dnd5e.yaml", Data: builtinGameSystemTemplateDnd5e},
	}
}

// BuiltinGameSystemTemplateAsset looks up a single embedded template by filename.
func BuiltinGameSystemTemplateAsset(filename string) (GameSystemTemplateAsset, bool) {
	for _, asset := range BuiltinGameSystemTemplateAssets() {
		if asset.Filename == filename {
			return asset, true
		}
	}
	return GameSystemTemplateAsset{}, false
}
