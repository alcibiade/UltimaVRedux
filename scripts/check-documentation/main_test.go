package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocumentationLinks(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*testing.T, string)
		want      string
		wantExact string
	}{
		{
			name: "valid local links and fragments",
			configure: func(t *testing.T, root string) {
				t.Helper()
				writeFixtureFile(t, root, "README.md", "# Project\n[encoded](docs/%47UIDE.md#repeated-1)\n[remote](https://example.com/missing)\n")
			},
		},
		{
			name: "valid local HTML targets",
			configure: func(t *testing.T, root string) {
				t.Helper()
				writeFixtureFile(t, root, "README.md", "# Project\n<a href='docs/GUIDE.md#guide'>Guide</a>\n<img alt=\"preview > image\" src=\"images/preview.png\">\n")
				writeFixtureFile(t, root, "images/preview.png", "image\n")
			},
		},
		{
			name: "missing target",
			configure: func(t *testing.T, root string) {
				t.Helper()
				writeFixtureFile(t, root, "README.md", "# Project\n[missing](docs/MISSING.md)\n")
			},
			want: "missing local Markdown target",
		},
		{
			name: "missing same-file fragment",
			configure: func(t *testing.T, root string) {
				t.Helper()
				writeFixtureFile(t, root, "README.md", "# Project\n[missing](#other)\n")
			},
			want: "missing Markdown heading fragment",
		},
		{
			name: "missing cross-file fragment",
			configure: func(t *testing.T, root string) {
				t.Helper()
				writeFixtureFile(t, root, "README.md", "# Project\n[missing](docs/GUIDE.md#other)\n")
			},
			want: "missing Markdown heading fragment",
		},
		{
			name: "duplicate heading requires numbered fragment",
			configure: func(t *testing.T, root string) {
				t.Helper()
				writeFixtureFile(t, root, "README.md", "# Project\n[missing](docs/GUIDE.md#repeated-2)\n")
			},
			want: "missing Markdown heading fragment",
		},
		{
			name: "root-relative DELETE_ME link",
			configure: func(t *testing.T, root string) {
				t.Helper()
				writeFixtureFile(t, root, "README.md", "# Project\n[deleted](/%44ELETE_ME/OLD.md)\n")
			},
			want: "links into DELETE_ME/",
		},
		{
			name: "leading slash local link",
			configure: func(t *testing.T, root string) {
				t.Helper()
				writeFixtureFile(t, root, "README.md", "# Project\n[guide](/docs/GUIDE.md)\n")
			},
			wantExact: "README.md:2: non-portable leading-slash local link: /docs/GUIDE.md",
		},
		{
			name: "traversal DELETE_ME link",
			configure: func(t *testing.T, root string) {
				t.Helper()
				writeFixtureFile(t, root, "docs/GUIDE.md", "# Guide\n[deleted](nested/../../DELETE_ME/OLD.md)\n")
			},
			want: "links into DELETE_ME/",
		},
		{
			name: "missing HTML target",
			configure: func(t *testing.T, root string) {
				t.Helper()
				writeFixtureFile(t, root, "README.md", "# Project\n<img src=\"images/missing.png\">\n")
			},
			wantExact: "README.md:2: missing local HTML target: images/missing.png",
		},
		{
			name: "HTML DELETE_ME target",
			configure: func(t *testing.T, root string) {
				t.Helper()
				writeFixtureFile(t, root, "README.md", "# Project\n<a href=\"DELETE_ME/OLD.md\">Old</a>\n")
			},
			wantExact: "README.md:2: retained document links into DELETE_ME/: DELETE_ME/OLD.md",
		},
		{
			name: "link-looking code is ignored",
			configure: func(t *testing.T, root string) {
				t.Helper()
				writeFixtureFile(t, root, "README.md", "# Project\n`[inline](missing-inline.md)` and `<img src=\"missing-inline.png\">`\n\n    [indented](missing-indented.md)\n    <a href=\"missing-indented.html\">missing</a>\n\n```md\n[fenced](missing-fenced.md)\n```\n")
			},
		},
		{
			name: "heading fragments are case sensitive",
			configure: func(t *testing.T, root string) {
				t.Helper()
				writeFixtureFile(t, root, "README.md", "# Project\n[guide](docs/GUIDE.md#Guide)\n")
			},
			wantExact: "README.md:2: missing Markdown heading fragment: docs/GUIDE.md#Guide",
		},
		{
			name: "globally unique heading collisions",
			configure: func(t *testing.T, root string) {
				t.Helper()
				writeFixtureFile(t, root, "README.md", "# Project\n[collision](docs/GUIDE.md#alpha-1-1)\n")
				writeFixtureFile(t, root, "docs/GUIDE.md", "# Guide\n\n## Alpha\n## ALPHA\n## alpha-1\n")
			},
		},
		{
			name: "GitHub-style markup emoji setext and CRLF headings",
			configure: func(t *testing.T, root string) {
				t.Helper()
				writeFixtureFile(t, root, "README.md", "# Project\r\n[markup](docs/GUIDE.md#-marked-link-tag-code)\r\n[setext](docs/GUIDE.md#setext-markup-)\r\n[windows](docs/GUIDE.md#windows-heading)\r\n")
				writeFixtureFile(t, root, "docs/GUIDE.md", "# Guide\r\n\r\n## 💥 *Marked* [Link](https://example.com) <em>Tag</em> `Code`\r\n\r\nSetext *Markup* 💥\r\n---\r\n\r\n## Windows Heading\r\n")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := newValidFixture(t)
			test.configure(t, root)
			findings := fixtureFindings(t, root)
			if test.wantExact != "" && findings != test.wantExact {
				t.Fatalf("findings %q, want %q", findings, test.wantExact)
			}
			if test.want == "" && findings != "" {
				if test.wantExact != "" {
					return
				}
				t.Fatalf("unexpected findings:\n%s", findings)
			}
			if test.want != "" && !strings.Contains(findings, test.want) {
				t.Fatalf("findings %q do not contain %q", findings, test.want)
			}
		})
	}
}

func TestLegacySaveCrosswalkRequiresEveryDecodedLeaf(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "internal/legacy_save/types.go", `package legacy_save
type Child struct { Value byte }
type SavedGame struct { Scalar byte; Children [2]Child }
type SavedObjects struct { Britannia [2]Child }
`)
	writeFixtureFile(
		t,
		root,
		"docs/details/LEGACY_TO_NATIVE_SAVE_CROSSWALK.md",
		"- **Exact:** `Scalar`\n- **Semantic:** `Children[].Value`\n",
	)

	findings, err := checkLegacySaveCrosswalk(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || !strings.Contains(findings[0].message, "`Objects.Britannia[].Value`") {
		t.Fatalf("findings = %#v", findings)
	}
}

func TestLegacySaveCrosswalkRequiresDispositionMarker(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "internal/legacy_save/types.go", `package legacy_save
type SavedGame struct { Scalar byte }
type SavedObjects struct { Britannia byte }
`)
	writeFixtureFile(
		t,
		root,
		"docs/details/LEGACY_TO_NATIVE_SAVE_CROSSWALK.md",
		"`Scalar` is mentioned without a disposition.\n- **Exact:** `Objects.Britannia`\n",
	)

	findings, err := checkLegacySaveCrosswalk(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || !strings.Contains(findings[0].message, "`Scalar`") {
		t.Fatalf("findings = %#v", findings)
	}
}

func TestLegacySaveCrosswalkRequiresTypeAuthorityWhenCrosswalkExists(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "docs/details/LEGACY_TO_NATIVE_SAVE_CROSSWALK.md", "# Crosswalk\n")

	findings, err := checkLegacySaveCrosswalk(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].path != "internal/legacy_save/types.go" {
		t.Fatalf("findings = %#v", findings)
	}
}

func TestDocumentationLinkSymlinks(t *testing.T) {
	tests := []struct {
		name       string
		targetPath func(*testing.T, string) string
		want       string
	}{
		{
			name: "outside repository",
			targetPath: func(t *testing.T, _ string) string {
				t.Helper()
				outsideRoot := t.TempDir()
				writeFixtureFile(t, outsideRoot, "outside.png", "outside\n")

				return filepath.Join(outsideRoot, "outside.png")
			},
			want: "README.md:2: local HTML link resolves outside repository root: images/linked.png",
		},
		{
			name: "DELETE_ME",
			targetPath: func(t *testing.T, root string) string {
				t.Helper()
				writeFixtureFile(t, root, "DELETE_ME/old.png", "old\n")

				return filepath.Join(root, "DELETE_ME", "old.png")
			},
			want: "README.md:2: retained document links into DELETE_ME/: images/linked.png",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := newValidFixture(t)
			writeFixtureFile(t, root, "README.md", "# Project\n<img src=\"images/linked.png\">\n")
			writeFixtureSymlink(t, root, "images/linked.png", test.targetPath(t, root))
			findings := fixtureFindings(t, root)
			if findings != test.want {
				t.Fatalf("findings %q, want %q", findings, test.want)
			}
		})
	}
}

func TestDocumentationCatalog(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*testing.T, string)
		want      string
	}{
		{
			name: "uncataloged document",
			configure: func(t *testing.T, root string) {
				t.Helper()
				writeFixtureFile(t, root, "docs/EXTRA.md", "# Extra\n")
			},
			want: "document missing from catalog: docs/EXTRA.md",
		},
		{
			name: "stale catalog entry",
			configure: func(t *testing.T, root string) {
				t.Helper()
				writeFixtureFile(t, root, "docs/README.md", validCatalog()+catalogRow("MISSING.md", "Missing", "owner", "Gamefact"))
			},
			want: "catalog references missing document: docs/MISSING.md",
		},
		{
			name: "duplicate catalog entry",
			configure: func(t *testing.T, root string) {
				t.Helper()
				writeFixtureFile(t, root, "docs/README.md", validCatalog()+catalogRow("GUIDE.md", "Duplicate", "owner", "Gamefact"))
			},
			want: "duplicate catalog entry: docs/GUIDE.md",
		},
		{
			name: "blank purpose",
			configure: func(t *testing.T, root string) {
				t.Helper()
				writeFixtureFile(t, root, "docs/README.md", catalogWithGuidePurpose(""))
			},
			want: "catalog purpose must not be blank",
		},
		{
			name: "invalid authority",
			configure: func(t *testing.T, root string) {
				t.Helper()
				writeFixtureFile(t, root, "docs/README.md", catalogWithAuthority("Historical"))
			},
			want: `invalid catalog authority "Historical"`,
		},
		{
			name: "invalid catalog row shape",
			configure: func(t *testing.T, root string) {
				t.Helper()
				catalog := strings.Replace(
					validCatalog(),
					"| [`GUIDE.md`](GUIDE.md) | Explain the system. | `owner` | Gamefact |",
					"| [`GUIDE.md`](GUIDE.md) | Explain the system. | `owner` |",
					1,
				)
				writeFixtureFile(t, root, "docs/README.md", catalog)
			},
			want: "catalog row must contain exactly four columns",
		},
		{
			name: "more than one owner",
			configure: func(t *testing.T, root string) {
				t.Helper()
				catalog := strings.Replace(validCatalog(), "`owner` | Gamefact", "`owner`, `other` | Gamefact", 1)
				writeFixtureFile(t, root, "docs/README.md", catalog)
			},
			want: "catalog row must name exactly one semantic domain owner",
		},
		{
			name: "blank domain owner",
			configure: func(t *testing.T, root string) {
				t.Helper()
				catalog := strings.Replace(validCatalog(), "`owner` | Gamefact", " | Gamefact", 1)
				writeFixtureFile(t, root, "docs/README.md", catalog)
			},
			want: "catalog row must name exactly one semantic domain owner",
		},
		{
			name: "invalid domain owner",
			configure: func(t *testing.T, root string) {
				t.Helper()
				catalog := strings.Replace(validCatalog(), "`owner` | Gamefact", "`../owner` | Gamefact", 1)
				writeFixtureFile(t, root, "docs/README.md", catalog)
			},
			want: "catalog row must name exactly one semantic domain owner",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := newValidFixture(t)
			test.configure(t, root)
			findings := fixtureFindings(t, root)
			if !strings.Contains(findings, test.want) {
				t.Fatalf("findings %q do not contain %q", findings, test.want)
			}
		})
	}
}

func TestCatalogTableCells(t *testing.T) {
	tests := []struct {
		name string
		row  string
		want []string
	}{
		{
			name: "escaped pipe",
			row:  `| Document | Purpose with an escaped \| pipe | Owner | Authority |`,
			want: []string{"Document", `Purpose with an escaped \| pipe`, "Owner", "Authority"},
		},
		{
			name: "pipe in code span",
			row:  "| Document | Purpose with `left | right` values | Owner | Authority |",
			want: []string{"Document", "Purpose with `left | right` values", "Owner", "Authority"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cells, ok := markdownTableCells(test.row)
			if !ok {
				t.Fatal("row was not recognized as a Markdown table row")
			}
			if strings.Join(cells, "\n") != strings.Join(test.want, "\n") {
				t.Fatalf("cells %#v, want %#v", cells, test.want)
			}
		})
	}
}

func TestCatalogTargetSymlinks(t *testing.T) {
	tests := []struct {
		name       string
		targetPath func(*testing.T, string) string
		want       finding
	}{
		{
			name: "outside repository",
			targetPath: func(t *testing.T, _ string) string {
				t.Helper()
				outsideRoot := t.TempDir()
				writeFixtureFile(t, outsideRoot, "OUTSIDE.md", "# Outside\n")

				return filepath.Join(outsideRoot, "OUTSIDE.md")
			},
			want: finding{"docs/README.md", 8, "catalog entry resolves outside repository root: LINK.md"},
		},
		{
			name: "DELETE_ME",
			targetPath: func(t *testing.T, root string) string {
				t.Helper()
				writeFixtureFile(t, root, "DELETE_ME/LINK.md", "# Old\n")

				return filepath.Join(root, "DELETE_ME", "LINK.md")
			},
			want: finding{"docs/README.md", 8, "catalog entry resolves into DELETE_ME/: LINK.md"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			root, err := filepath.EvalSymlinks(root)
			if err != nil {
				t.Fatalf("canonicalize fixture root: %v", err)
			}
			catalog := "# Documentation Catalog\n\n## General\n\n" +
				"| Document | One-line purpose | Primary owner | Authority classification |\n" +
				"|---|---|---|---|\n" +
				catalogRow("README.md", "Catalog retained docs.", "owner", "Index") +
				catalogRow("LINK.md", "Linked document.", "owner", "Gamefact")
			writeFixtureFile(t, root, "docs/README.md", catalog)
			writeFixtureSymlink(t, root, "docs/LINK.md", test.targetPath(t, root))

			findings, err := checkCatalog(root, []string{"docs/README.md", "docs/LINK.md"})
			if err != nil {
				t.Fatalf("check catalog: %v", err)
			}
			if !containsFinding(findings, test.want) {
				t.Fatalf("findings %#v do not contain %#v", findings, test.want)
			}
		})
	}
}

func TestCatalogDomainOwnerNeedsNoExternalManifest(t *testing.T) {
	root := newValidFixture(t)
	catalog := strings.ReplaceAll(validCatalog(), "`owner`", "`save-system`")
	writeFixtureFile(t, root, "docs/README.md", catalog)
	if findings := fixtureFindings(t, root); findings != "" {
		t.Fatalf("public-only catalog has unexpected findings: %s", findings)
	}
}

func TestObsoleteDocumentationDirectory(t *testing.T) {
	for _, directory := range []string{"AI_IGNORE", "migration", "historical", "history", "stubs", "gamefacts/stubs"} {
		t.Run(directory, func(t *testing.T) {
			root := newValidFixture(t)
			writeFixtureFile(t, root, "docs/"+directory+"/OLD.txt", "obsolete\n")
			findings := fixtureFindings(t, root)
			want := "docs/" + directory + ": obsolete documentation directory contains files"
			if findings != want {
				t.Fatalf("findings %q, want %q", findings, want)
			}
		})
	}
}

func newValidFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFixtureFile(t, root, "README.md", "# Project\n[guide](docs/GUIDE.md#guide)\n")
	writeFixtureFile(t, root, "docs/README.md", validCatalog())
	writeFixtureFile(t, root, "docs/GUIDE.md", "# Guide\n\n[project](../README.md#project)\n[self](#repeated-1)\n[catalog](README.md#documentation-catalog)\n\n## Repeated\n## Repeated\n")

	return root
}

func validCatalog() string {
	return "# Documentation Catalog\n\n## General\n\n" +
		"| Document | One-line purpose | Primary owner | Authority classification |\n" +
		"|---|---|---|---|\n" +
		catalogRow("README.md", "Catalog retained docs.", "owner", "Index") +
		catalogRow("GUIDE.md", "Explain the system.", "owner", "Gamefact")
}

func catalogWithGuidePurpose(purpose string) string {
	return "# Documentation Catalog\n\n## General\n\n" +
		"| Document | One-line purpose | Primary owner | Authority classification |\n" +
		"|---|---|---|---|\n" +
		catalogRow("README.md", "Catalog retained docs.", "owner", "Index") +
		catalogRow("GUIDE.md", purpose, "owner", "Gamefact")
}

func catalogWithAuthority(authority string) string {
	return "# Documentation Catalog\n\n## General\n\n" +
		"| Document | One-line purpose | Primary owner | Authority classification |\n" +
		"|---|---|---|---|\n" +
		catalogRow("README.md", "Catalog retained docs.", "owner", "Index") +
		catalogRow("GUIDE.md", "Explain the system.", "owner", authority)
}

func catalogRow(document, purpose, owner, authority string) string {
	return "| [`" + document + "`](" + document + ") | " + purpose + " | `" + owner + "` | " + authority + " |\n"
}

func writeFixtureFile(t *testing.T, root, relativePath, content string) {
	t.Helper()
	absolutePath := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(absolutePath, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}
}

func writeFixtureSymlink(t *testing.T, root, relativePath, target string) {
	t.Helper()
	absolutePath := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o755); err != nil {
		t.Fatalf("create symlink fixture directory: %v", err)
	}
	if err := os.Symlink(target, absolutePath); err != nil {
		t.Skipf("symlinks unavailable on this platform: %v", err)
	}
}

func containsFinding(findings []finding, want finding) bool {
	for _, item := range findings {
		if item == want {
			return true
		}
	}

	return false
}

func fixtureFindings(t *testing.T, root string) string {
	t.Helper()
	findings, err := checkRepository(root)
	if err != nil {
		t.Fatalf("check fixture: %v", err)
	}
	lines := make([]string, len(findings))
	for index, item := range findings {
		lines[index] = item.String()
	}

	return strings.Join(lines, "\n")
}
