package templates

// ServerDecodeTemplate is the templates for generating the server-side decoding
// function for a particular Binding.
var ServerDecodeTemplate = `
{{- with $binding := . -}}
	// DecodeHTTP{{$binding.Label}}Request is a transport/http.DecodeRequestFunc that
	// decodes a JSON-encoded {{ToLower $binding.Parent.Name}} request from the HTTP request
	// body. Primarily useful in a server.
	func DecodeHTTP{{$binding.Label}}Request(_ context.Context, r *http.Request) (interface{}, error) {
		var req {{GoPackageName $binding.Parent.Request.Name}}.{{GoName $binding.Parent.Request.Name}}

		// to support gzip input
		var reader io.ReadCloser
		var err error 
		switch r.Header.Get("Content-Encoding") {
		case "gzip":
			reader, err = gzip.NewReader(r.Body)
			defer reader.Close()
			if err != nil {
				return nil, nhttp.WrapError(err, 400, "failed to read the gzip content")
			}
		default:
			reader = r.Body
		}

		buf, err := ioutil.ReadAll(reader)
		if err != nil {
			return nil, nhttp.WrapError(err, 400, "cannot read body of http request")
		}
		if len(buf) > 0 {
			{{- if not $binding.Body}}
			if err = jsoniter.ConfigFastest.Unmarshal(buf, &req); err != nil {
			{{else}}
			req.{{GoName $binding.Body.Name}} = {{if $binding.Body.IsMap}}make({{$binding.Body.GetGoTypeName}}){{else}}&{{$binding.Body.GetGoTypeName}}{}{{end}}
			if err = jsoniter.ConfigFastest.Unmarshal(buf, req.{{GoName $binding.Body.Field.Name}}); err != nil {
			{{end -}}
				const size = 8196
				if len(buf) > size {
					buf = buf[:size]
				}
				return nil, nhttp.WrapError(err,
					http.StatusBadRequest,
					fmt.Sprintf("request body '%s': cannot parse non-json request body", buf),
				)
			}
		}

		pathParams := mux.Vars(r)
		_ = pathParams

		queryParams := r.URL.Query()
		_ = queryParams

		parsedQueryParams := make(map[string]bool)
		_ = parsedQueryParams

		{{range $param := $binding.Parameters}}
			{{if ne $param.Location "body"}}
				{{$param.Go.QueryUnmarshaler}}
			{{end}}
		{{end}}
		return &req, err
	}
{{- end -}}
`

// WARNING: Changing the contents of these strings, even a little bit, will cause tests
// to fail. So don't change them purely because you think they look a little
// funny.

var ServerTemplate = `// Code generated by ncraft. DO NOT EDIT.
// Rerunning ncraft will overwrite this file.
// Version: {{.Version}}
// Version Date: {{.VersionDate}}

package svc

// This file provides server-side bindings for the HTTP transport.
// It utilizes the transport/http.Server.

import (
	"bytes"
	"encoding/json"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"io"

	"context"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/json-iterator/go"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/tracing/opentracing"

	httptransport "github.com/go-kit/kit/transport/http"
	pagination "github.com/ncraft-io/ncraft-gokit/pkg/pagination"
	nhttp "github.com/ncraft-io/ncraft-gokit/pkg/transport/http"
	stdopentracing "github.com/opentracing/opentracing-go"

    {{$corePackage := "github.com/mojo-lang/core/go/pkg/mojo/core"}}
    "{{$corePackage}}"
    {{range $i := .Go.ImportedTypePaths}}
	{{if ne $i $corePackage}}"{{$i}}"{{end}}
	{{- end}}

	// this service api
	pb "{{.Go.ApiImportPath -}}"
)

const contentType = "application/json; charset=utf-8"

var (
	_ = fmt.Sprint
	_ = bytes.Compare
	_ = strconv.Atoi
	_ = httptransport.NewServer
	_ = ioutil.NopCloser
	_ = pb.New{{.Interface.BaredName}}Client
	_ = io.Copy
	_ = errors.Wrap
)
{{if .HasImported}}
var ({{range $msg := .ImportedMessages}}
	_ = {{$msg.Go.PackageName}}.{{$msg.Name}}{}
{{- end}}{{range $enum := .ImportedEnums}}
	_ = {{$enum.Go.PackageName}}.{{$enum.Name}}(0)
{{- end}}){{end}}

// RegisterHttpHandler register a set of endpoints available on predefined paths to the router.
func RegisterHttpHandler(router *mux.Router, endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger)  {
	{{- if .Interface.Methods}}
		serverOptions := []httptransport.ServerOption{
			httptransport.ServerBefore(headersToContext),
			httptransport.ServerErrorEncoder(errorEncoder),
			httptransport.ServerErrorLogger(logger),
			httptransport.ServerAfter(httptransport.SetContentType(contentType)),
		}
	{{- end }}

	addTracerOption := func(methodName string) []httptransport.ServerOption {
	    if tracer != nil {
	        return append(serverOptions, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, methodName, logger)))
	    }
	    return serverOptions
	}

	{{range $method := .Interface.Methods}}
		{{range $binding := $method.Bindings}}
			router.Methods("{{$binding.Verb | ToUpper}}").Path("{{$binding.Path}}").Handler(
				httptransport.NewServer(
					endpoints.{{ToCamel $method.Name}}Endpoint,
					DecodeHTTP{{$binding.Label}}Request,{{if $binding.GetResponseBody}}
					EncodeHTTP{{ToCamel $method.Name}}Response,{{else}}
					EncodeHTTPGenericResponse,{{end}}
					addTracerOption("{{$method.Name}}")...,
					//append(serverOptions, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "{{ToSnake $method.Name}}", logger)))...,
			))
		{{- end}}
	{{- end}}
}

// ErrorEncoder writes the error to the ResponseWriter, by default a content
// type of application/json, a body of json with key "error" and the value
// error.Error(), and a status code of 500. If the error implements Headerer,
// the provided headers will be applied to the response. If the error
// implements json.Marshaler, and the marshaling succeeds, the JSON encoded
// form of the error will be used. If the error implements StatusCoder, the
// provided StatusCode will be used instead of 500.
func errorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	body, _ := json.Marshal(errorWrapper{Error: err.Error()})
	if marshaler, ok := err.(json.Marshaler); ok {
		if jsonBody, marshalErr := marshaler.MarshalJSON(); marshalErr == nil {
			body = jsonBody
		}
	}
    e, ok := err.(*core.Error)
    if !ok {
        e = core.NewErrorFrom(500, err.Error())
	}

    if jsonBody, marshalErr := jsoniter.ConfigFastest.Marshal(e); marshalErr == nil {
        body = jsonBody
    }

	w.Header().Set("Content-Type", contentType)
	if headerer, ok := err.(httptransport.Headerer); ok {
		for k := range headerer.Headers() {
			w.Header().Set(k, headerer.Headers().Get(k))
		}
	}
	code := http.StatusInternalServerError
	if sc, ok := err.(httptransport.StatusCoder); ok {
		code = sc.StatusCode()
	}
	w.WriteHeader(code)
	w.Write(body)
}

type errorWrapper struct {
	Error string ` + "`" + `json:"error"` + "`" + `
}

// Server Decode
{{range $method := .Interface.Methods}}
	{{range $binding := $method.Bindings}}
		{{$binding.Extensions.ServerDecode}}
	{{end}}
{{end}}

{{range $method := .Interface.Methods}}
    {{range $binding := $method.Bindings}}
	    {{if $binding.GetResponseBody}}
func EncodeHTTP{{ToCamel $method.Name}}Response(_ context.Context, w http.ResponseWriter, response interface{}) error {
	r := response.(*{{GoPackageName $method.Response.Body.Name}}.{{$method.Response.Body.Name}})

	{{range $h, $v := $method.Response.Headers}}
	w.Header().Set("{{$h}}", "{{$v}}")
	{{end}}

	cnt, err := w.Write([]byte(r.{{$method.Response.Body.Name}}))
	if err != nil {
		return err
	}
	responseCnt := len(r.{{$method.Response.Body.Name}})
	if cnt != responseCnt {
		return errors.Errorf("wrong to write response content expect %d, but %d written", responseCnt, cnt)
	}

	return nil
}
        {{end}}
    {{end}}
{{end}}

// EncodeHTTPGenericResponse is a transport/http.EncodeResponseFunc that encodes
// the response as JSON to the response writer. Primarily useful in a server.
func EncodeHTTPGenericResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if _, ok := response.(*core.Null); ok {
		return nil 
	}

    if p, ok := response.(pagination.Paginater); ok {
		total := p.GetTotalCount()
		if total > 0 {
			w.Header().Set("X-Total-Count", strconv.Itoa(int(total)))
		}

		next := p.GetNextPageToken()
		if len(next) > 0 {
			path, _ := ctx.Value("http-request-path").(string)
			if len(path) == 0 {
				path = "/?next-page-token=" + next
			} else {
				url, _ := core.ParseUrl(path)
				url.Query.Add("next-page-token", next)
				path = url.Format()
			}
			w.Header().Set("Link", fmt.Sprintf("<%s>; rel=\"next\"", path, next))
		}
	}

	encoder := jsoniter.ConfigFastest.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(response)
}

// Helper functions

func headersToContext(ctx context.Context, r *http.Request) context.Context {
	for k, _ := range r.Header {
		// The key is added both in http format (k) which has had
		// http.CanonicalHeaderKey called on it in transport as well as the
		// strings.ToLower which is the grpc metadata format of the key so
		// that it can be accessed in either format
		ctx = context.WithValue(ctx, k, r.Header.Get(k))
		ctx = context.WithValue(ctx, strings.ToLower(k), r.Header.Get(k))
	}

	// add the access key to context
	accessKey := r.URL.Query().Get("access_key")
	if len(accessKey) > 0{
		ctx = context.WithValue(ctx, "access_key", accessKey)
	}

	// Tune specific change.
	// also add the request url
	ctx = context.WithValue(ctx, "http-request-path", r.URL.Path)
	ctx = context.WithValue(ctx, "transport", "HTTPJSON")

	return ctx
}

// json array value: []
// delimiter based string: ,
func ParseArrayStr(str string, delimiter string) []string {
	if len(str) == 0 {
		return []string{}
	}

	str = strings.TrimSpace(str)
	if str[0] == '[' {
		str = str[1:len(str)-1]
	}

	return strings.Split(str, delimiter)
}
`
