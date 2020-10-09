package reviser

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"io/ioutil"
	"sort"
	"strings"

	"github.com/incu6us/goimports-reviser/pkg/astutil"
	"github.com/incu6us/goimports-reviser/pkg/std"
)

// Revise imports and format the code
func Execute(projectName, filePath string, optionList ...Option) ([]byte, bool, error) {
	options := applyOptions(optionList)

	originalContent, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, false, err
	}

	fset := token.NewFileSet()

	pf, err := parser.ParseFile(fset, "", originalContent, parser.ParseComments)
	if err != nil {
		return nil, false, err
	}

	importsWithMetadata := combineAllImportsWithMetadata(pf, options)

	importGroups := groupImports(projectName, importsWithMetadata, options)

	fixImports(pf, importGroups, importsWithMetadata)

	fixedImportsContent, err := generateFile(fset, pf)
	if err != nil {
		return nil, false, err
	}

	formattedContent, err := format.Source(fixedImportsContent)
	if err != nil {
		return nil, false, err
	}

	return formattedContent, !bytes.Equal(originalContent, formattedContent), nil
}

func groupImports(projectName string, importsWithMetadata map[string]*commentsMetadata, options *Options) [][]string {
	var (
		stdImports     []string
		projectImports []string
		generalImports []string
	)
	extraGroupImports := map[string][]string{}

	for imprt := range importsWithMetadata {
		pkgWithoutAlias := skipPackageAlias(imprt)

		if _, ok := std.StdPackages[pkgWithoutAlias]; ok {
			stdImports = append(stdImports, imprt)
			continue
		}

		if strings.Contains(pkgWithoutAlias, projectName) {
			projectImports = append(projectImports, imprt)
			continue
		}

		if set := func() bool {
			for _, group := range options.ExtraImportGroups {
				if strings.Contains(pkgWithoutAlias, group) {
					extraGroupImports[group] = append(extraGroupImports[group], imprt)
					return true
				}
			}
			return false
		}(); set {
			continue
		}

		generalImports = append(generalImports, imprt)
	}

	sort.Strings(stdImports)
	sort.Strings(generalImports)
	sort.Strings(projectImports)
	for _, group := range extraGroupImports {
		sort.Strings(group)
	}

	importGroups := [][]string{}
	importGroups = append(importGroups, stdImports)
	importGroups = append(importGroups, generalImports)
	for _, group := range options.ExtraImportGroups {
		if len(extraGroupImports[group]) != 0 {
			importGroups = append(importGroups, extraGroupImports[group])
		}
	}
	importGroups = append(importGroups, projectImports)

	return importGroups
}

func skipPackageAlias(pkg string) string {
	values := strings.Split(pkg, " ")
	if len(values) > 1 {
		return values[1]
	}

	return pkg
}

func generateFile(fset *token.FileSet, file *ast.File) ([]byte, error) {
	var output []byte
	buffer := bytes.NewBuffer(output)
	if err := printer.Fprint(buffer, fset, file); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func fixImports(f *ast.File, importGroups [][]string, commentsMetadata map[string]*commentsMetadata) {
	var importsPositions []*importPosition

	for _, decl := range f.Decls {
		switch decl.(type) {
		case *ast.GenDecl:
			dd := decl.(*ast.GenDecl)
			if dd.Tok == token.IMPORT {
				importsPositions = append(importsPositions, &importPosition{
					Start: dd.Pos(),
					End:   dd.End(),
				})

				var specs []ast.Spec

				first := true
				for _, group := range importGroups {
					if !first && len(group) > 0 {
						spec := &ast.ImportSpec{Path: &ast.BasicLit{Value: "", Kind: token.STRING}}
						specs = append(specs, spec)
					}

					linesCounter := len(group)
					for _, imprt := range group {
						spec := &ast.ImportSpec{
							Path: &ast.BasicLit{Value: importWithComment(imprt, commentsMetadata), Kind: dd.Tok},
						}
						specs = append(specs, spec)

						linesCounter--
					}
					first = false
				}
				dd.Specs = specs
			}
		}
	}

	clearImportDocs(f, importsPositions)
}

func clearImportDocs(f *ast.File, importsPositions []*importPosition) {
	importsComments := make([]*ast.CommentGroup, 0, len(f.Comments))

	for _, comment := range f.Comments {
		for _, importPosition := range importsPositions {
			if importPosition.IsInRange(comment) {
				continue
			}
			importsComments = append(importsComments, comment)
		}
	}

	if len(f.Imports) > 0 {
		f.Comments = importsComments
	}
}

func importWithComment(imprt string, commentsMetadata map[string]*commentsMetadata) string {
	var comment string
	commentGroup, ok := commentsMetadata[imprt]
	if ok {
		if commentGroup != nil && commentGroup.Comment != nil && len(commentGroup.Comment.List) > 0 {
			comment = fmt.Sprintf("// %s", commentGroup.Comment.Text())
		}
	}

	return fmt.Sprintf("%s %s", imprt, comment)
}

func combineAllImportsWithMetadata(f *ast.File, options *Options) map[string]*commentsMetadata {
	importsWithMetadata := map[string]*commentsMetadata{}

	for _, decl := range f.Decls {
		switch decl.(type) {
		case *ast.GenDecl:
			dd := decl.(*ast.GenDecl)
			if dd.Tok == token.IMPORT {
				for _, spec := range dd.Specs {
					var importSpecStr string
					importSpec := spec.(*ast.ImportSpec)

					if options.RemoveUnusedImports && !astutil.UsesImport(f, strings.Trim(importSpec.Path.Value, `"`)) {
						continue
					}

					if importSpec.Name != nil {
						importSpecStr = strings.Join([]string{importSpec.Name.String(), importSpec.Path.Value}, " ")
					} else {
						if options.AliasForVersionSuffix {
							importSpecStr = setAliasForVersionedImportSpec(importSpec)
						} else {
							importSpecStr = importSpec.Path.Value
						}
					}

					importsWithMetadata[importSpecStr] = &commentsMetadata{
						Doc:     importSpec.Doc,
						Comment: importSpec.Comment,
					}
				}
			}
		}
	}
	return importsWithMetadata
}

func setAliasForVersionedImportSpec(importSpec *ast.ImportSpec) string {
	var importSpecStr string

	aliasName, ok := astutil.PackageNameFromImportPath(strings.Trim(importSpec.Path.Value, `"`))
	if ok {
		importSpecStr = fmt.Sprintf("%s %s", aliasName, importSpec.Path.Value)
	} else {
		importSpecStr = importSpec.Path.Value
	}

	return importSpecStr
}

type commentsMetadata struct {
	Doc     *ast.CommentGroup
	Comment *ast.CommentGroup
}

type importPosition struct {
	Start token.Pos
	End   token.Pos
}

func (p *importPosition) IsInRange(comment *ast.CommentGroup) bool {
	if p.Start <= comment.Pos() && comment.Pos() <= p.End {
		return true
	}

	return false
}
