package compiler

import (
	"github.com/iancoleman/strcase"
	"github.com/mojo-lang/lang/go/pkg/mojo/lang"
	"strings"
)

type BoxedDictionary struct {
	PackageName   string
	GoPackageName string
	Name          string
	FullName      string
	EnclosingName string
	FieldName     string
}

func (b *BoxedDictionary) Compile(decl *lang.StructDecl) error {
	b.PackageName = decl.GetPackageName()
	b.GoPackageName = GetGoPackage(decl.GetPackageName())
	b.Name = decl.Name
	b.EnclosingName = strings.Join(lang.GetEnclosingNames(decl.EnclosingType), ".")
	b.FullName = GetFullName(b.EnclosingName, b.Name)
	b.FieldName = strcase.ToCamel(decl.Type.Fields[0].Name)
	return nil
}
