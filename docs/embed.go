// Package docs carries the documentation tree into the binary: the map
// (nav.yaml) and every page of its groups, in English and in every
// translation (ru/: its map ru/nav.yaml and the same pages under the same
// paths). Design records (plans/), the assets and the generated llms files
// stay out, and a page the map lists outside this directory
// (../CONTRIBUTING.md) is not carried either. internal/docs is what reads
// it; nothing else should.
package docs

import "embed"

// FS holds nav.yaml and the Markdown pages of the documentation groups, and
// the same for the Russian and Chinese translations. A new group directory has
// to be added to the pattern below in every tree: the tests of internal/docs
// fail when a page of the map is missing from any of them.
//
//go:embed nav.yaml getting-started/*.md surfaces/*.md operate/*.md features/*.md reference/*.md tutorials/*.md contributing/*.md
//go:embed ru/nav.yaml ru/getting-started/*.md ru/surfaces/*.md ru/operate/*.md ru/features/*.md ru/reference/*.md ru/tutorials/*.md ru/contributing/*.md
//go:embed zh/nav.yaml zh/getting-started/*.md zh/surfaces/*.md zh/operate/*.md zh/features/*.md zh/reference/*.md zh/tutorials/*.md zh/contributing/*.md
var FS embed.FS
