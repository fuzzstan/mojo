package compiler

import (
	"github.com/mojo-lang/core/go/pkg/mojo/core"
	"github.com/mojo-lang/document/go/pkg/mojo/document"
	"github.com/mojo-lang/openapi/go/pkg/mojo/openapi"
)

type OperationCompiler struct {
	Components *openapi.Components
}

//### 请求参数
//
//#### Path 参数
//
//#### Query 参数
//
//| 参数 | 类型 | 是否必须 | 默认值 | 说明 |
//|:---  |:----|:-----   | :---- | :----|
//{{range .TableList}}| {{.Index}}|[ {{.TableName}}](#{{.Index}}{{.TableName}})       |{{.Comment}}| |
//{{end}}
//
//#### Body 请求对象
//
//##### Body 请求示例
//
//{{ range .Schemas}}
//{{end}}
//
//#### 完整请求示例
//
//### 返回值
//
//#### 返回对象
//
//{{ range .Schemas}}
//#### {{}}
//
//| 参数 | 类型 | 是否必须 | 默认值 | 说明 |
//|:---  |:----|:-----   | :---- | :----|
//{{range .TableList}}| {{.Index}}|[ {{.TableName}}](#{{.Index}}{{.TableName}})       |{{.Comment}}| |
//{{end}}
//
//#####
//
//{{end}}
//
//### API示例

func (o *OperationCompiler) Compile(operation *openapi.Operation) (*document.Document, error) {
	doc := &document.Document{}
	ctx := &Context{}

	doc.AppendHeaderFromText(3, "请求参数")

	parameters := operation.GetLocationParameters()
	if ps := parameters[openapi.Parameter_LOCATION_PATH]; len(ps) > 0 {
		doc.AppendHeaderFromText(4, "Path 参数")
		err := o.compilePathParameters(ctx, ps, doc)
		if err != nil {
			return nil, err
		}
	}

	if ps := parameters[openapi.Parameter_LOCATION_QUERY]; len(ps) > 0 {
		doc.AppendHeaderFromText(4, "Query 参数")
		err := o.compileQueryParameters(ctx, ps, doc)
		if err != nil {
			return nil, err
		}
	}

	if operation.GetRequestBody() != nil {
		doc.AppendHeaderFromText(4, "Body 请求对象")
		err := o.compileRequestBody(ctx, operation.GetRequestBody().GetRequestBody(), doc)
		if err != nil {
			return nil, err
		}
	}

	doc.AppendHeaderFromText(3, "返回值")
	if resp, ok := operation.Responses.Values["200"]; ok {
		response := resp.GetResponse()
		doc.AppendHeaderFromText(4, "返回对象")
		if len(response.Content) == 0 {
			doc.AppendBlock(document.NewTextPlainBlock("对象为空"))
		} else {
			err := o.compileResponse(ctx, response, doc)
			if err != nil {
				return nil, err
			}
		}
	}

	return doc, nil
}

func (o *OperationCompiler) compilePathParameters(ctx *Context, parameters []*openapi.Parameter, doc *document.Document) error {
	table := &document.Table{
		Caption:   nil,
		Alignment: 0,
		Header: document.NewTextTableHeader("参数名", "参数类型", "格式类型", "说明"),
	}

	for _, parameter := range parameters {
		row := &document.Table_Row{}

		row.Values = append(row.Values, document.NewTableCell(wrapCodeToBlock(parameter.Name)))

		typeName := parameter.Schema.GetTypeName(o.Components.Schemas)
		row.Values = append(row.Values, document.NewTableCell(wrapCodeToBlock(typeName)))

		typeFormat := parameter.Schema.GetFormat(o.Components.Schemas)
		row.Values = append(row.Values, document.NewTableCell(wrapCodeToBlock(typeFormat))) // 格式类型

		row.Values = append(row.Values, document.NewTextTableCell(parameter.Description))

		table.Rows = append(table.Rows, row)
	}

	doc.AppendTable(table)
	return nil
}

func (o *OperationCompiler) compileQueryParameters(ctx *Context, parameters []*openapi.Parameter, doc *document.Document) error {
	table := &document.Table{
		Caption:   nil,
		Alignment: 0,
		Header: document.NewTextTableHeader("参数名", "参数类型", "格式类型", "是否必须", "默认值", "说明"),
	}

	for _, parameter := range parameters {
		row := &document.Table_Row{}

		row.Values = append(row.Values, document.NewTableCell(wrapCodeToBlock(parameter.Name)))

		typeName := parameter.Schema.GetTypeName(o.Components.Schemas)
		row.Values = append(row.Values, document.NewTableCell(wrapCodeToBlock(typeName)))

		typeFormat := parameter.Schema.GetFormat(o.Components.Schemas)
		row.Values = append(row.Values, document.NewTableCell(wrapCodeToBlock(typeFormat))) // 格式类型

		required := "否"
		if parameter.Required {
			required = "是"
		}
		row.Values = append(row.Values, document.NewTextTableCell(required)) // 是否必须

		row.Values = append(row.Values, document.NewTextTableCell("")) // 默认值

		row.Values = append(row.Values, document.NewTextTableCell(parameter.Description))

		table.Rows = append(table.Rows, row)
	}

	doc.AppendTable(table)
	return nil
}

func (o *OperationCompiler) compileRequestBody(ctx *Context, body *openapi.RequestBody, doc *document.Document) error {
	if body == nil || doc == nil {
		return nil
	}

	compiler := &SchemaCompiler{Components: o.Components}
	if mediaType, ok := body.Content[core.ApplicationJson]; ok {
		schema := mediaType.Schema.GetSchemaOf(o.Components.Schemas)
		document, err := compiler.Compile(schema)
		if err != nil {
			return err
		}
		doc.Blocks = append(doc.Blocks, document.Blocks...)

		deps := schema.Dependencies(o.Components.Schemas)
		for _, dep := range deps {
			document, err = compiler.Compile(dep)
			if err != nil {
				return err
			}
			doc.AppendHeaderFrom(4, wrapCode(dep.Title))
			doc.Blocks = append(doc.Blocks, document.Blocks...)
		}
	}
	return nil
}

func (o *OperationCompiler) compileResponse(ctx *Context, response *openapi.Response, doc *document.Document) error {
	if response == nil || doc == nil {
		return nil
	}

	compiler := &SchemaCompiler{Components: o.Components}
	if mediaType, ok := response.Content[core.ApplicationJson]; ok {
		schema := mediaType.Schema.GetSchemaOf(o.Components.Schemas)
		document, err := compiler.Compile(schema)
		if err != nil {
			return err
		}
		doc.Blocks = append(doc.Blocks, document.Blocks...)

		deps := schema.Dependencies(o.Components.Schemas)
		for _, dep := range deps {
			document, err = compiler.Compile(dep)
			if err != nil {
				return err
			}
			doc.AppendHeaderFrom(4, wrapCode(dep.Title))
			doc.Blocks = append(doc.Blocks, document.Blocks...)
		}
	}

	return nil
}

func (o *OperationCompiler) compileErrorResponse(ctx *Context) {
}
