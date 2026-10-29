// Package templates contains the application shipped with this CLI version.
package templates

import "embed"

// Go source and go.mod use .tmpl suffixes so package discovery sees only this
// embedding package. Generation removes the suffixes. Explicit patterns include
// dotfiles and workflows without embedding this file into generated projects.
//
//go:embed .env.example .gitignore all:.github Makefile README.md compose.yaml go.mod.tmpl go.sum all:api all:cmd all:configs all:domain all:internal all:migrations all:pkg
var Files embed.FS
