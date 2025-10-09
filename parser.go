package tsgen

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Config contains parser configuration.
type Config struct {
	// Mapping maps dependencies to specific types.
	Mapping map[TypeReference]Type
}

// Parser parses Go packages.
type Parser struct {
	// Config contains parser configuration.
	Config Config
}

// NewParser creates a new Parser.
func NewParser(cfg Config) *Parser {
	return &Parser{
		Config: cfg,
	}
}

// Parse parses the given package names.
func (p *Parser) Parse(pkgNames ...string) ([]*Package, error) {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedSyntax |
			packages.NeedTypesInfo,
	}, pkgNames...)
	if err != nil {
		return nil, fmt.Errorf("loading packages: %w", err)
	}

	pkgParsers := make([]*packageParser, 0, len(pkgs))
	for _, pkg := range pkgs {
		pkgParsers = append(pkgParsers, p.newPackageParsers(pkg)...)
	}

	outPkgs := make([]*Package, 0, len(pkgParsers))
	for _, pkgParser := range pkgParsers {
		outPkg, err := pkgParser.Parse()
		if err != nil {
			return nil, fmt.Errorf("parsing package: %w", err)
		}
		outPkgs = append(outPkgs, outPkg)
	}

	return outPkgs, nil
}

func (p *Parser) newPackageParsers(pkg *packages.Package) []*packageParser {
	if isStdLib(pkg.PkgPath) {
		return nil
	}

	parsers := make([]*packageParser, 0, 1+len(pkg.Imports))
	parsers = append(parsers, newPackageParser(p.Config, pkg))

	for _, imp := range pkg.Imports {
		parsers = append(parsers, p.newPackageParsers(imp)...)
	}

	return parsers
}

type packageParser struct {
	cfg Config
	pkg *packages.Package
	out *Package
}

func newPackageParser(cfg Config, pkg *packages.Package) *packageParser {
	return &packageParser{
		pkg: pkg,
		cfg: cfg,
	}
}

func (p *packageParser) Parse() (*Package, error) {
	if len(p.pkg.Errors) > 0 {
		return nil, fmt.Errorf("loading package: %+v", p.pkg.Errors)
	}
	if len(p.pkg.GoFiles) == 0 {
		return nil, fmt.Errorf("no Go files in package %s", p.pkg.Name)
	}

	p.out = &Package{
		Name: p.pkg.PkgPath,
	}

	for _, file := range p.pkg.Syntax {
		p.parseFile(file)
	}

	return p.out, nil
}

func (p *packageParser) parseFile(file *ast.File) {
	ast.Inspect(file, func(node ast.Node) bool {
		switch nodeType := node.(type) {
		case *ast.GenDecl:
			p.parseGenDecl(nodeType)
			return false
		default:
			return true
		}
	})
}

func (p *packageParser) parseGenDecl(decl *ast.GenDecl) {
	for _, spec := range decl.Specs {
		switch specType := spec.(type) {
		case *ast.TypeSpec:
			p.parseTypeSpec(specType)

		case *ast.ValueSpec:
			switch decl.Tok {
			case token.VAR:
				//panic("not implemented")

			case token.CONST:
				//p.parseConstSpec(specType)

			default:
				//panic("not implemented")
			}

		default:
			//panic("not implemented")
		}
	}
}

func (p *packageParser) parseTypeSpec(spec *ast.TypeSpec) {
	switch typeExpr := spec.Type.(type) {
	case *ast.Ident:
		// `type Foobar string`
		p.parseTypeExpr(spec, typeExpr)

	case *ast.StructType:
		// `type Foobar struct {}`
		p.parseTypeStruct(spec, typeExpr)

	case *ast.InterfaceType:
		// Ignore `type Foobar interface {}`.

	case *ast.MapType:
		// `type Foobar map[string]Baz`
		p.parseTypeExpr(spec, typeExpr)

	case *ast.FuncType:
		// Ignore `type Foobar func() {}`.

	case *ast.ArrayType:
		// `type Foobar []Baz`
		p.parseTypeExpr(spec, typeExpr)

	case *ast.SelectorExpr:
		// `type Foobar time.Time`
		p.parseTypeExpr(spec, typeExpr)

	case *ast.ChanType:
		// `type Foobar chan Baz`
		p.parseTypeExpr(spec, typeExpr)

	case *ast.StarExpr:
		p.parseTypeExpr(spec, typeExpr)

	default:
		panic("not implemented")
	}
}

func (p *packageParser) Name(expr ast.Expr) string {
	switch exprType := expr.(type) {
	case *ast.Ident:
		return exprType.Name
	default:
		panic("not implemented")
	}
}

func (p *packageParser) parseTypeStruct(spec *ast.TypeSpec, expr *ast.StructType) {
	name := p.Name(spec.Name)
	if !ast.IsExported(name) {
		return
	}

	node := &NodeStruct{
		Name: name,
		Doc:  p.Doc(spec.Doc),
	}

	for _, f := range expr.Fields.List {
		tags := p.parseTags(f.Tag)
		if tags.JSON("-") {
			continue
		}

		if f.Names == nil || tags.JSON("inline") {
			node.Fields = append(node.Fields, &Field{
				Doc:   p.Doc(f.Doc),
				Type:  p.parseType(f.Type),
				Tags:  tags,
				Embed: true,
			})
			continue
		}

		name = p.Names(f.Names...)
		if !ast.IsExported(name) {
			continue
		}

		node.Fields = append(node.Fields, &Field{
			Name: name,
			Doc:  p.Doc(f.Doc),
			Type: p.parseType(f.Type),
			Tags: tags,
		})
	}

	p.out.Nodes = append(p.out.Nodes, node)
}

func (p *packageParser) Doc(doc *ast.CommentGroup) string {
	return strings.TrimSpace(doc.Text())
}

func (p *packageParser) Names(idents ...*ast.Ident) string {
	names := make([]string, 0, len(idents))
	for _, ident := range idents {
		names = append(names, ident.Name)
	}
	return strings.Join(names, ".")
}

func (p *packageParser) parseType(expr ast.Expr) Type {
	switch exprType := expr.(type) {
	case *ast.Ident:
		return TypePrimitive{Name: exprType.Name}

	case *ast.StarExpr:
		return TypePointer{Type: p.parseType(exprType.X)}

	case *ast.SelectorExpr:
		ref := TypeReference{
			PkgPath: p.findPackagePath(exprType),
			Name:    p.Name(exprType.Sel),
		}
		if mapped, ok := p.cfg.Mapping[ref]; ok {
			return mapped
		}
		return ref

	case *ast.ArrayType:
		return TypeArray{Type: p.parseType(exprType.Elt)}

	case *ast.MapType:
		return TypeMap{
			Key:   p.parseType(exprType.Key),
			Value: p.parseType(exprType.Value),
		}

	case *ast.InterfaceType:
		return TypePrimitive{Name: "any"}

	case *ast.FuncType:
		return TypeUnimplemented{Name: "func"}

	case *ast.IndexExpr:
		return TypeUnimplemented{Name: "index"}

	default:
		panic("not implemented")
	}
}

func (p *packageParser) findPackagePath(sel *ast.SelectorExpr) string {
	if x, ok := sel.X.(*ast.Ident); ok {
		if obj := p.pkg.TypesInfo.Uses[x]; obj != nil {
			if pkgName, ok := obj.(*types.PkgName); ok {
				return pkgName.Imported().Path()
			}
		}
	}

	panic("not implemented")
}

func (p *packageParser) parseTags(tag *ast.BasicLit) Tags {
	if tag == nil {
		return nil
	}

	res := make(Tags, 1)
	for _, elem := range strings.Split(strings.Trim(tag.Value, "`"), " ") {
		key, value, found := strings.Cut(elem, ":")
		if !found {
			continue
		}
		res[key] = strings.Trim(value, `"`)
	}
	return res
}

func (p *packageParser) parseTypeExpr(spec *ast.TypeSpec, expr ast.Expr) {
	name := p.Name(spec.Name)
	if !ast.IsExported(name) {
		return
	}

	p.out.Nodes = append(p.out.Nodes, &NodeType{
		Name: name,
		Doc:  p.Doc(spec.Doc),
		Type: p.parseType(expr),
	})
}

func isStdLib(path string) bool {
	return !strings.Contains(path, ".")
}
