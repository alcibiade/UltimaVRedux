// Command check-documentation validates retained Markdown links, ownership, and authority rules.
package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"html"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

var (
	errLegacySaveRootTypesMissing = errors.New("legacy save types must define SavedGame and SavedObjects")
	fencePattern                  = regexp.MustCompile(`^ {0,3}(` + "`{3,}" + `|~{3,})`)
	inlineLinkPattern             = regexp.MustCompile(`!?\[[^\]\n]*\]\(([^)\n]+)\)`)
	htmlAttributePattern          = regexp.MustCompile(
		`(?i)(?:^|[ \t\r\n])(?:href|src)[ \t\r\n]*=[ \t\r\n]*(?:"([^"]*)"|'([^']*)'|([^ \t\r\n"'=<>]+))`,
	)
	atxHeadingPattern  = regexp.MustCompile(`^ {0,3}#{1,6}[ \t]+(.+?)(?:[ \t]+#+[ \t]*)?$`)
	setextPattern      = regexp.MustCompile(`^ {0,3}(?:=+|-+)[ \t]*$`)
	inlineLabelPattern = regexp.MustCompile(`!?\[([^\]]+)\]\([^)]+\)`)
	htmlTagPattern     = regexp.MustCompile(`<[^>]+>`)
	ownerPattern       = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

type finding struct {
	path    string
	line    int
	message string
}

func (item finding) String() string {
	if item.line == 0 {
		return fmt.Sprintf("%s: %s", item.path, item.message)
	}

	return fmt.Sprintf("%s:%d: %s", item.path, item.line, item.message)
}

type localTarget struct {
	path         string
	fragment     string
	remote       bool
	leadingSlash bool
}

type linkReference struct {
	destination string
	offset      int
	kind        string
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "determine repository root: %v\n", err)
		os.Exit(1)
	}

	findings, err := checkRepository(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, item := range findings {
		fmt.Fprintln(os.Stderr, item)
	}
	if len(findings) > 0 {
		os.Exit(1)
	}
}

func checkRepository(root string) ([]finding, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve repository root: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("canonicalize repository root: %w", err)
	}

	markdownFiles, err := retainedMarkdownFiles(root)
	if err != nil {
		return nil, err
	}

	var findings []finding
	anchorCache := make(map[string]map[string]bool)
	for _, relativePath := range markdownFiles {
		absolutePath := filepath.Join(root, filepath.FromSlash(relativePath))
		content, readErr := os.ReadFile(absolutePath) //nolint:gosec // Paths come from the fixed retained roots.
		if readErr != nil {
			return nil, fmt.Errorf("read %q: %w", relativePath, readErr)
		}
		text := string(content)

		linkFindings, linkErr := checkLinks(root, relativePath, text, anchorCache)
		if linkErr != nil {
			return nil, linkErr
		}
		findings = append(findings, linkFindings...)
	}

	catalogFindings, err := checkCatalog(root, markdownFiles)
	if err != nil {
		return nil, err
	}
	findings = append(findings, catalogFindings...)
	legacyCoverageFindings, err := checkLegacySaveCrosswalk(root)
	if err != nil {
		return nil, err
	}
	findings = append(findings, legacyCoverageFindings...)

	obsoleteFindings, err := checkObsoleteDirectories(root)
	if err != nil {
		return nil, err
	}
	findings = append(findings, obsoleteFindings...)

	return uniqueSortedFindings(findings), nil
}

func checkLegacySaveCrosswalk(root string) ([]finding, error) {
	typesPath := filepath.Join(root, "internal", "legacy_save", "types.go")
	crosswalkPath := filepath.Join(root, "docs", "details", "LEGACY_TO_NATIVE_SAVE_CROSSWALK.md")
	if _, err := os.Stat(typesPath); os.IsNotExist(err) {
		if _, crosswalkErr := os.Stat(crosswalkPath); crosswalkErr == nil {
			return []finding{{
				path: "internal/legacy_save/types.go", line: 0,
				message: "missing decoded legacy type authority for save crosswalk",
			}}, nil
		}

		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("inspect legacy save types: %w", err)
	}
	content, err := os.ReadFile(crosswalkPath) //nolint:gosec // Repository-owned fixed path.
	if err != nil {
		if os.IsNotExist(err) {
			return []finding{{
				path: "docs/details/LEGACY_TO_NATIVE_SAVE_CROSSWALK.md", line: 0,
				message: "missing legacy save field crosswalk",
			}}, nil
		}

		return nil, fmt.Errorf("read legacy save crosswalk: %w", err)
	}
	keys, err := legacySaveCoverageKeys(typesPath)
	if err != nil {
		return nil, err
	}
	findings := make([]finding, 0)
	lines := strings.Split(string(content), "\n")
	for _, key := range keys {
		if !legacySaveKeyHasDisposition(lines, key) {
			findings = append(findings, finding{
				path:    "docs/details/LEGACY_TO_NATIVE_SAVE_CROSSWALK.md",
				line:    0,
				message: "missing decoded legacy field disposition for `" + key + "`",
			})
		}
	}

	return findings, nil
}

func legacySaveKeyHasDisposition(lines []string, key string) bool {
	dispositions := [...]string{"**Exact:**", "**Semantic:**", "**Derived/defaulted:**", "**Omitted:**"}
	for _, line := range lines {
		if !strings.Contains(line, "`"+key+"`") {
			continue
		}
		for _, disposition := range dispositions {
			if strings.Contains(line, disposition) {
				return true
			}
		}
	}

	return false
}

func legacySaveCoverageKeys(typesPath string) ([]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), typesPath, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse legacy save types: %w", err)
	}
	structs := make(map[string]*ast.StructType)
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.TYPE {
			continue
		}
		for _, specification := range general.Specs {
			typeSpec, ok := specification.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if structure, ok := typeSpec.Type.(*ast.StructType); ok {
				structs[typeSpec.Name.Name] = structure
			}
		}
	}
	keys := make([]string, 0)
	if structs["SavedGame"] == nil || structs["SavedObjects"] == nil {
		return nil, errLegacySaveRootTypesMissing
	}
	collectLegacyStructKeys(structs, "SavedGame", "", &keys)
	collectLegacyStructKeys(structs, "SavedObjects", "Objects", &keys)
	sort.Strings(keys)

	return keys, nil
}

func collectLegacyStructKeys(structs map[string]*ast.StructType, typeName, prefix string, keys *[]string) {
	structure := structs[typeName]
	if structure == nil {
		return
	}
	for _, field := range structure.Fields.List {
		for _, name := range field.Names {
			path := name.Name
			if prefix != "" {
				path = prefix + "." + path
			}
			switch fieldType := field.Type.(type) {
			case *ast.Ident:
				if _, nested := structs[fieldType.Name]; nested {
					collectLegacyStructKeys(structs, fieldType.Name, path, keys)
				} else {
					*keys = append(*keys, path)
				}
			case *ast.ArrayType:
				if element, ok := fieldType.Elt.(*ast.Ident); ok {
					if _, nested := structs[element.Name]; nested {
						collectLegacyStructKeys(structs, element.Name, path+"[]", keys)

						continue
					}
				}
				*keys = append(*keys, path)
			default:
				*keys = append(*keys, path)
			}
		}
	}
}

func retainedMarkdownFiles(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read repository root: %w", err)
	}

	var paths []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			paths = append(paths, entry.Name())
		}
	}

	roots := []string{"docs"}
	for _, relativeRoot := range roots {
		absoluteRoot := filepath.Join(root, filepath.FromSlash(relativeRoot))
		if _, statErr := os.Stat(absoluteRoot); os.IsNotExist(statErr) {
			continue
		} else if statErr != nil {
			return nil, fmt.Errorf("inspect %q: %w", relativeRoot, statErr)
		}

		walkErr := filepath.WalkDir(absoluteRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".md") {
				return nil
			}
			relativePath, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return fmt.Errorf("make %q repository-relative: %w", path, relErr)
			}
			paths = append(paths, filepath.ToSlash(relativePath))

			return nil
		})
		if walkErr != nil {
			return nil, fmt.Errorf("scan %q: %w", relativeRoot, walkErr)
		}
	}

	sort.Strings(paths)

	return paths, nil
}

//nolint:cyclop,funlen // Each branch reports one independent local-link invariant.
func checkLinks(
	root string,
	relativePath string,
	text string,
	anchorCache map[string]map[string]bool,
) ([]finding, error) {
	prose := withoutMarkdownCode(text)
	references := markdownLinkReferences(prose)
	references = append(references, htmlLinkReferences(prose)...)
	findings := make([]finding, 0)
	for _, reference := range references {
		destination := reference.destination
		target, err := resolveLocalTarget(root, relativePath, destination)
		line := lineNumber(prose, reference.offset)
		if err != nil {
			findings = append(findings, finding{
				relativePath,
				line,
				fmt.Sprintf("invalid %s %q: %v", reference.kind, destination, err),
			})

			continue
		}
		if target.remote {
			continue
		}

		targetRelative, insideRoot := repositoryRelative(root, target.path)
		if !insideRoot {
			findings = append(findings, finding{
				relativePath,
				line,
				"local " + reference.kind + " escapes repository root: " + destination,
			})

			continue
		}
		if isDeleteMePath(targetRelative) {
			findings = append(findings, finding{
				relativePath,
				line,
				"retained document links into DELETE_ME/: " + destination,
			})

			continue
		}
		if target.leadingSlash {
			findings = append(findings, finding{
				relativePath,
				line,
				"non-portable leading-slash local link: " + destination,
			})

			continue
		}

		info, statErr := os.Stat(target.path)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				findings = append(findings, finding{
					relativePath,
					line,
					"missing local " + strings.TrimSuffix(reference.kind, " link") + " target: " + destination,
				})

				continue
			}

			return nil, fmt.Errorf("inspect Markdown target %q from %q: %w", target.path, relativePath, statErr)
		}
		resolvedPath, resolveErr := filepath.EvalSymlinks(target.path)
		if resolveErr != nil {
			return nil, fmt.Errorf("resolve local target %q from %q: %w", target.path, relativePath, resolveErr)
		}
		if containmentFinding, invalid := resolvedLinkContainmentFinding(
			root,
			resolvedPath,
			relativePath,
			line,
			reference.kind,
			destination,
		); invalid {
			findings = append(findings, containmentFinding)

			continue
		}
		if target.fragment == "" || info.IsDir() || !strings.EqualFold(filepath.Ext(target.path), ".md") {
			continue
		}

		anchors, found := anchorCache[resolvedPath]
		if !found {
			anchors, err = markdownAnchors(resolvedPath)
			if err != nil {
				return nil, err
			}
			anchorCache[resolvedPath] = anchors
		}
		if !anchors[target.fragment] {
			findings = append(findings, finding{
				relativePath,
				line,
				"missing Markdown heading fragment: " + destination,
			})
		}
	}

	return findings, nil
}

func resolvedLinkContainmentFinding(
	root string,
	resolvedPath string,
	relativePath string,
	line int,
	kind string,
	destination string,
) (finding, bool) {
	resolvedRelative, insideRoot := repositoryRelative(root, resolvedPath)
	if !insideRoot {
		return finding{
			relativePath,
			line,
			"local " + kind + " resolves outside repository root: " + destination,
		}, true
	}
	if isDeleteMePath(resolvedRelative) {
		return finding{
			relativePath,
			line,
			"retained document links into DELETE_ME/: " + destination,
		}, true
	}

	return finding{path: "", line: 0, message: ""}, false
}

func markdownLinkReferences(text string) []linkReference {
	matches := inlineLinkPattern.FindAllStringSubmatchIndex(text, -1)
	references := make([]linkReference, 0, len(matches))
	for _, match := range matches {
		references = append(references, linkReference{
			destination: markdownDestination(text[match[2]:match[3]]),
			offset:      match[0],
			kind:        "Markdown link",
		})
	}

	return references
}

func htmlLinkReferences(text string) []linkReference {
	var references []linkReference
	for tagStart := 0; tagStart < len(text); {
		relativeStart := strings.IndexByte(text[tagStart:], '<')
		if relativeStart < 0 {
			break
		}
		tagStart += relativeStart
		if tagStart+1 >= len(text) || !isASCIILetter(text[tagStart+1]) {
			tagStart++

			continue
		}

		tagEnd := htmlTagEnd(text, tagStart+1)
		if tagEnd < 0 {
			break
		}
		tag := text[tagStart : tagEnd+1]
		for _, match := range htmlAttributePattern.FindAllStringSubmatchIndex(tag, -1) {
			for capture := 2; capture < len(match); capture += 2 {
				if match[capture] < 0 {
					continue
				}
				references = append(references, linkReference{
					destination: html.UnescapeString(tag[match[capture]:match[capture+1]]),
					offset:      tagStart + match[capture],
					kind:        "HTML link",
				})

				break
			}
		}
		tagStart = tagEnd + 1
	}

	return references
}

func htmlTagEnd(text string, start int) int {
	quote := byte(0)
	for index := start; index < len(text); index++ {
		character := text[index]
		if quote != 0 {
			if character == quote {
				quote = 0
			}

			continue
		}
		switch character {
		case '\'', '"':
			quote = character
		case '>':
			return index
		}
	}

	return -1
}

func isASCIILetter(character byte) bool {
	return character >= 'A' && character <= 'Z' || character >= 'a' && character <= 'z'
}

func markdownDestination(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "<") {
		if end := strings.Index(raw, ">"); end >= 0 {
			return raw[1:end]
		}
	}
	if fields := strings.Fields(raw); len(fields) > 0 {
		return fields[0]
	}

	return raw
}

func resolveLocalTarget(root, sourceRelative, destination string) (localTarget, error) {
	if strings.HasPrefix(destination, "//") {
		return localTarget{path: "", fragment: "", remote: true, leadingSlash: false}, nil
	}

	parsed, err := url.Parse(destination)
	if err != nil {
		return localTarget{}, fmt.Errorf("parse destination: %w", err)
	}
	if parsed.IsAbs() || parsed.Host != "" {
		return localTarget{path: "", fragment: "", remote: true, leadingSlash: false}, nil
	}

	decodedPath, err := url.PathUnescape(parsed.EscapedPath())
	if err != nil {
		return localTarget{}, fmt.Errorf("decode path: %w", err)
	}
	fragment, err := url.PathUnescape(parsed.EscapedFragment())
	if err != nil {
		return localTarget{}, fmt.Errorf("decode fragment: %w", err)
	}

	source := filepath.Join(root, filepath.FromSlash(sourceRelative))
	var targetPath string
	switch {
	case decodedPath == "":
		targetPath = source
	case strings.HasPrefix(decodedPath, "/"):
		targetPath = filepath.Join(root, filepath.FromSlash(strings.TrimLeft(decodedPath, "/")))
	default:
		targetPath = filepath.Join(filepath.Dir(source), filepath.FromSlash(decodedPath))
	}

	return localTarget{
		path:         filepath.Clean(targetPath),
		fragment:     fragment,
		remote:       false,
		leadingSlash: strings.HasPrefix(decodedPath, "/"),
	}, nil
}

func repositoryRelative(root, target string) (string, bool) {
	relativePath, err := filepath.Rel(root, target)
	if err != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return "", false
	}

	return filepath.ToSlash(relativePath), true
}

func isDeleteMePath(relativePath string) bool {
	return relativePath == "DELETE_ME" || strings.HasPrefix(relativePath, "DELETE_ME/")
}

func markdownAnchors(path string) (map[string]bool, error) {
	content, err := os.ReadFile(path) //nolint:gosec // Paths come from repository-contained Markdown targets.
	if err != nil {
		return nil, fmt.Errorf("read Markdown headings from %q: %w", path, err)
	}

	lines := strings.Split(withoutMarkdownBlockCode(string(content)), "\n")
	anchors := make(map[string]bool)
	for index, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		heading := ""
		if match := atxHeadingPattern.FindStringSubmatch(line); match != nil {
			heading = match[1]
		} else if index+1 < len(lines) && strings.TrimSpace(line) != "" &&
			setextPattern.MatchString(strings.TrimSuffix(lines[index+1], "\r")) {
			heading = strings.TrimSpace(line)
		}
		if heading == "" {
			continue
		}

		baseSlug := githubHeadingSlug(heading)
		slug := baseSlug
		for suffix := 1; anchors[slug]; suffix++ {
			slug = fmt.Sprintf("%s-%d", baseSlug, suffix)
		}
		anchors[slug] = true
	}

	return anchors, nil
}

func githubHeadingSlug(heading string) string {
	heading = inlineLabelPattern.ReplaceAllString(heading, "$1")
	heading = html.UnescapeString(htmlTagPattern.ReplaceAllString(heading, ""))
	heading = strings.ToLower(strings.ReplaceAll(heading, "`", ""))

	var slug strings.Builder
	for _, character := range heading {
		switch {
		case unicode.IsLetter(character), unicode.IsNumber(character), unicode.IsMark(character),
			character == '_', character == '-':
			slug.WriteRune(character)
		case character == ' ':
			slug.WriteByte('-')
		}
	}

	return slug.String()
}

func withoutMarkdownCode(text string) string {
	masked := []byte(withoutMarkdownBlockCode(text))
	for index := 0; index < len(masked); {
		if masked[index] != '`' || escapedByte(masked, index) {
			index++

			continue
		}
		delimiterEnd := index
		for delimiterEnd < len(masked) && masked[delimiterEnd] == '`' {
			delimiterEnd++
		}
		delimiterLength := delimiterEnd - index
		closingStart := matchingBacktickRun(masked, delimiterEnd, delimiterLength)
		if closingStart < 0 {
			index = delimiterEnd

			continue
		}
		maskBytes(masked, index, closingStart+delimiterLength)
		index = closingStart + delimiterLength
	}

	return string(masked)
}

func withoutMarkdownBlockCode(text string) string {
	masked := []byte(text)
	inFence := false
	fenceCharacter := byte(0)
	fenceLength := 0
	for lineStart := 0; lineStart < len(masked); {
		lineEnd := lineStart
		for lineEnd < len(masked) && masked[lineEnd] != '\n' {
			lineEnd++
		}
		line := strings.TrimSuffix(string(masked[lineStart:lineEnd]), "\r")
		match := fencePattern.FindStringSubmatch(line)
		if match != nil {
			marker := match[1]
			if !inFence {
				inFence = true
				fenceCharacter = marker[0]
				fenceLength = len(marker)
			} else if marker[0] == fenceCharacter && len(marker) >= fenceLength {
				inFence = false
			}
			maskBytes(masked, lineStart, lineEnd)
		} else if inFence || isIndentedCodeLine(line) {
			maskBytes(masked, lineStart, lineEnd)
		}
		lineStart = lineEnd + 1
	}

	return string(masked)
}

func isIndentedCodeLine(line string) bool {
	return strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "    ")
}

func matchingBacktickRun(text []byte, start, delimiterLength int) int {
	for index := start; index < len(text); {
		if text[index] != '`' || escapedByte(text, index) {
			index++

			continue
		}
		runEnd := index
		for runEnd < len(text) && text[runEnd] == '`' {
			runEnd++
		}
		if runEnd-index == delimiterLength {
			return index
		}
		index = runEnd
	}

	return -1
}

func escapedByte(text []byte, index int) bool {
	backslashes := 0
	for index--; index >= 0 && text[index] == '\\'; index-- {
		backslashes++
	}

	return backslashes%2 == 1
}

func maskBytes(text []byte, start, end int) {
	for index := start; index < end; index++ {
		if text[index] != '\n' && text[index] != '\r' {
			text[index] = ' '
		}
	}
}

func lineNumber(text string, offset int) int {
	return strings.Count(text[:offset], "\n") + 1
}

//nolint:cyclop,funlen,gocognit,gocyclo // The catalog table is one schema whose fields need precise findings.
func checkCatalog(root string, markdownFiles []string) ([]finding, error) {
	const (
		catalogRelative    = "docs/README.md"
		catalogColumnCount = 4
	)

	catalogPath := filepath.Join(root, filepath.FromSlash(catalogRelative))
	content, err := os.ReadFile(catalogPath) //nolint:gosec // The catalog path is a fixed repository location.
	if err != nil {
		return nil, fmt.Errorf("read documentation catalog: %w", err)
	}

	documents := make(map[string]bool)
	for _, relativePath := range markdownFiles {
		if strings.HasPrefix(relativePath, "docs/") {
			documents[relativePath] = true
		}
	}

	entries := make(map[string]int)
	var findings []finding
	lines := strings.Split(string(content), "\n")
	inCatalogTable := false
	for index, line := range lines {
		cells, tableRow := markdownTableCells(line)
		if tableRow && len(cells) == catalogColumnCount && strings.EqualFold(cells[0], "Document") {
			inCatalogTable = true

			continue
		}
		if !inCatalogTable {
			continue
		}
		if !tableRow {
			inCatalogTable = false

			continue
		}
		if isTableSeparator(cells) {
			continue
		}
		if len(cells) != catalogColumnCount {
			findings = append(findings, finding{
				catalogRelative,
				index + 1,
				"catalog row must contain exactly four columns",
			})

			continue
		}

		purpose := strings.TrimSpace(cells[1])
		ownerCell := strings.TrimSpace(cells[2])
		authority := strings.TrimSpace(cells[3])
		if purpose == "" {
			findings = append(findings, finding{catalogRelative, index + 1, "catalog purpose must not be blank"})
		}
		owner := strings.Trim(ownerCell, "`")
		if ownerCell != "`"+owner+"`" || !ownerPattern.MatchString(owner) {
			findings = append(findings, finding{
				catalogRelative,
				index + 1,
				"catalog row must name exactly one semantic domain owner",
			})
		}
		if !allowedAuthority(authority) {
			findings = append(findings, finding{
				catalogRelative,
				index + 1,
				fmt.Sprintf("invalid catalog authority %q", authority),
			})
		}

		destination, destinationOK := catalogDestination(cells[0])
		if !destinationOK {
			findings = append(findings, finding{catalogRelative, index + 1, "catalog document must be one local Markdown link"})

			continue
		}
		target, resolveErr := resolveLocalTarget(root, catalogRelative, destination)
		if resolveErr != nil || target.remote || target.fragment != "" || target.leadingSlash {
			findings = append(findings, finding{
				catalogRelative,
				index + 1,
				"invalid catalog document target: " + destination,
			})

			continue
		}
		targetRelative, insideRoot := repositoryRelative(root, target.path)
		if !insideRoot ||
			!strings.HasPrefix(targetRelative, "docs/") ||
			!strings.EqualFold(filepath.Ext(targetRelative), ".md") {
			findings = append(findings, finding{
				catalogRelative,
				index + 1,
				"catalog entry escapes docs/: " + destination,
			})

			continue
		}
		if info, statErr := os.Stat(target.path); statErr == nil {
			resolvedPath, evalErr := filepath.EvalSymlinks(target.path)
			if evalErr != nil {
				return nil, fmt.Errorf("resolve catalog target %q: %w", target.path, evalErr)
			}
			resolvedRelative, resolvedInsideRoot := repositoryRelative(root, resolvedPath)
			switch {
			case !resolvedInsideRoot:
				findings = append(findings, finding{
					catalogRelative,
					index + 1,
					"catalog entry resolves outside repository root: " + destination,
				})
			case isDeleteMePath(resolvedRelative):
				findings = append(findings, finding{
					catalogRelative,
					index + 1,
					"catalog entry resolves into DELETE_ME/: " + destination,
				})
			case info.IsDir() || !strings.HasPrefix(resolvedRelative, "docs/"):
				findings = append(findings, finding{
					catalogRelative,
					index + 1,
					"catalog entry resolves outside docs/: " + destination,
				})
			}
			if !resolvedInsideRoot || isDeleteMePath(resolvedRelative) || info.IsDir() ||
				!strings.HasPrefix(resolvedRelative, "docs/") {
				continue
			}
		} else if !os.IsNotExist(statErr) {
			return nil, fmt.Errorf("inspect catalog target %q: %w", target.path, statErr)
		}

		entries[targetRelative]++
		if entries[targetRelative] > 1 {
			findings = append(findings, finding{
				catalogRelative,
				index + 1,
				"duplicate catalog entry: " + targetRelative,
			})
		}
	}

	for document := range documents {
		if entries[document] == 0 {
			findings = append(findings, finding{
				catalogRelative,
				0,
				"document missing from catalog: " + document,
			})
		}
	}
	for entry := range entries {
		if !documents[entry] {
			findings = append(findings, finding{
				catalogRelative,
				0,
				"catalog references missing document: " + entry,
			})
		}
	}

	return findings, nil
}

func allowedAuthority(authority string) bool {
	switch authority {
	case "Index", "General policy", "System documentation", "Procedure", "ADR rationale", "Gamefact":
		return true
	default:
		return false
	}
}

func markdownTableCells(line string) ([]string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "|") || !strings.HasSuffix(trimmed, "|") {
		return nil, false
	}

	body := strings.TrimSuffix(strings.TrimPrefix(trimmed, "|"), "|")
	parts := []string{""}
	codeDelimiterLength := 0
	for index := 0; index < len(body); {
		switch body[index] {
		case '\\':
			index = appendEscapedTableCharacter(parts, body, index)
		case '`':
			index, codeDelimiterLength = appendTableCodeDelimiter(parts, body, index, codeDelimiterLength)
		case '|':
			if codeDelimiterLength == 0 {
				parts = append(parts, "")
			} else {
				parts[len(parts)-1] += "|"
			}
			index++
		default:
			parts[len(parts)-1] += string(body[index])
			index++
		}
	}
	for index := range parts {
		parts[index] = strings.TrimSpace(parts[index])
	}

	return parts, true
}

func appendEscapedTableCharacter(parts []string, body string, index int) int {
	parts[len(parts)-1] += string(body[index])
	index++
	if index < len(body) {
		parts[len(parts)-1] += string(body[index])
		index++
	}

	return index
}

func appendTableCodeDelimiter(parts []string, body string, index, codeDelimiterLength int) (int, int) {
	runEnd := index
	for runEnd < len(body) && body[runEnd] == '`' {
		runEnd++
	}
	runLength := runEnd - index
	parts[len(parts)-1] += body[index:runEnd]
	switch codeDelimiterLength {
	case 0:
		codeDelimiterLength = runLength
	case runLength:
		codeDelimiterLength = 0
	}

	return runEnd, codeDelimiterLength
}

func isTableSeparator(cells []string) bool {
	if len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		if strings.Trim(cell, "-:") != "" || !strings.Contains(cell, "-") {
			return false
		}
	}

	return true
}

func catalogDestination(cell string) (string, bool) {
	match := inlineLinkPattern.FindStringSubmatch(strings.TrimSpace(cell))
	if match == nil || match[0] != strings.TrimSpace(cell) {
		return "", false
	}

	return markdownDestination(match[1]), true
}

func checkObsoleteDirectories(root string) ([]finding, error) {
	obsoleteNames := map[string]bool{
		"AI_IGNORE":  true,
		"historical": true,
		"history":    true,
		"migration":  true,
		"stubs":      true,
	}
	var findings []finding
	docsRoot := filepath.Join(root, "docs")
	err := filepath.WalkDir(docsRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() || path == docsRoot || !obsoleteNames[entry.Name()] {
			return nil
		}

		containsFile, scanErr := directoryContainsFile(path)
		if scanErr != nil {
			return fmt.Errorf("inspect obsolete directory %q: %w", path, scanErr)
		}
		if containsFile {
			relativeDirectory, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return fmt.Errorf("make obsolete directory repository-relative: %w", relErr)
			}
			relativeDirectory = filepath.ToSlash(relativeDirectory)
			findings = append(findings, finding{relativeDirectory, 0, "obsolete documentation directory contains files"})
		}

		return filepath.SkipDir
	})
	if err != nil {
		return nil, fmt.Errorf("scan obsolete documentation directories: %w", err)
	}

	return findings, nil
}

func directoryContainsFile(root string) (bool, error) {
	containsFile := false
	err := filepath.WalkDir(root, func(_ string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			containsFile = true
		}

		return nil
	})
	if err != nil {
		return false, fmt.Errorf("scan directory: %w", err)
	}

	return containsFile, nil
}

func uniqueSortedFindings(findings []finding) []finding {
	sort.Slice(findings, func(left, right int) bool {
		if findings[left].path != findings[right].path {
			return findings[left].path < findings[right].path
		}
		if findings[left].line != findings[right].line {
			return findings[left].line < findings[right].line
		}

		return findings[left].message < findings[right].message
	})

	unique := findings[:0]
	for _, item := range findings {
		if len(unique) == 0 || item != unique[len(unique)-1] {
			unique = append(unique, item)
		}
	}

	return unique
}
