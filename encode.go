package gosoap

import (
	"encoding/xml"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

var (
	soapPrefix                            = "soap"
	customEnvelopeAttrs map[string]string = nil
)

// SetCustomEnvelope define customizated envelope
func SetCustomEnvelope(prefix string, attrs map[string]string) {
	soapPrefix = prefix
	if attrs != nil {
		customEnvelopeAttrs = attrs
	}
}

// MarshalXML envelope the body and encode to xml
func (c process) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	tokens := &tokenData{}

	//start envelope
	if c.Client.Definitions == nil {
		return fmt.Errorf("definitions is nil")
	}

	namespace := ""
	// 1. 检查 Types 是否存在且不为空
	if c.Client.Definitions.Types != nil && len(c.Client.Definitions.Types) > 0 {
		// 2. 检查该 Type 下是否有 Schema 定义
		if len(c.Client.Definitions.Types[0].XsdSchema) > 0 {
			schema := c.Client.Definitions.Types[0].XsdSchema[0]
			namespace = c.Client.Definitions.TargetNamespace

			// 3. 检查 Imports 是否存在
			if namespace == "" && len(schema.Imports) > 0 {
				namespace = schema.Imports[0].Namespace
			}
		} else {
			// 如果没有 Schema，回退到使用 TargetNamespace
			namespace = c.Client.Definitions.TargetNamespace
		}
	}

	tokens.startEnvelope()
	if c.Client.HeaderParams != nil {
		tokens.startHeader(c.Client.HeaderName, namespace)
		tokens.recursiveEncode(c.Client.HeaderParams)
		tokens.endHeader(c.Client.HeaderName)
	}

	err := tokens.startBody(c.Request.Method, namespace)
	if err != nil {
		return err
	}

	tokens.recursiveEncode(c.Request.Params)

	//end envelope
	tokens.endBody(c.Request.Method)
	tokens.endEnvelope()

	for _, t := range tokens.data {
		err := e.EncodeToken(t)
		if err != nil {
			return err
		}
	}

	return e.Flush()
}

type tokenData struct {
	data []xml.Token
}

func (tokens *tokenData) recursiveEncode(hm interface{}) {
	v := reflect.ValueOf(hm)

	switch v.Kind() {
	case reflect.Map:
		for _, key := range v.MapKeys() {
			t := xml.StartElement{
				Name: xml.Name{
					Space: "",
					Local: key.String(),
				},
			}

			tokens.data = append(tokens.data, t)
			tokens.recursiveEncode(v.MapIndex(key).Interface())
			tokens.data = append(tokens.data, xml.EndElement{Name: t.Name})
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			tokens.recursiveEncode(v.Index(i).Interface())
		}
	case reflect.Array:
		if v.Len() == 2 {
			label := v.Index(0).Interface()
			t := xml.StartElement{
				Name: xml.Name{
					Space: "",
					Local: label.(string),
				},
			}

			tokens.data = append(tokens.data, t)
			tokens.recursiveEncode(v.Index(1).Interface())
			tokens.data = append(tokens.data, xml.EndElement{Name: t.Name})
		}
	case reflect.String:
		content := xml.CharData(v.String())
		tokens.data = append(tokens.data, content)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Fix: Add support for int types
		content := xml.CharData(strconv.FormatInt(v.Int(), 10))
		tokens.data = append(tokens.data, content)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// Fix: Add support for uint types
		content := xml.CharData(strconv.FormatUint(v.Uint(), 10))
		tokens.data = append(tokens.data, content)
	case reflect.Float32, reflect.Float64:
		// Fix: Add support for float types
		content := xml.CharData(strconv.FormatFloat(v.Float(), 'f', -1, 64))
		tokens.data = append(tokens.data, content)
	case reflect.Bool:
		// Fix: Add support for bool type
		content := xml.CharData(strconv.FormatBool(v.Bool()))
		tokens.data = append(tokens.data, content)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			var name string
			name = v.Type().Field(i).Tag.Get("xml")
			if name == "" {
				name = v.Type().Field(i).Name
				name = strings.ToLower(name[0:1]) + name[1:]
			}
			t := xml.StartElement{
				Name: xml.Name{
					Space: "",
					Local: name,
				},
			}
			tokens.data = append(tokens.data, t)
			tokens.recursiveEncode(field.Interface())
			tokens.data = append(tokens.data, xml.EndElement{Name: t.Name})
		}
	default:
		// Fix: Handle other types by converting to string
		content := xml.CharData(fmt.Sprintf("%v", v.Interface()))
		tokens.data = append(tokens.data, content)
	}
}

func (tokens *tokenData) startEnvelope() {
	e := xml.StartElement{
		Name: xml.Name{
			Space: "",
			Local: fmt.Sprintf("%s:Envelope", soapPrefix),
		},
	}

	if customEnvelopeAttrs == nil {
		e.Attr = []xml.Attr{
			{Name: xml.Name{Space: "", Local: "xmlns:xsi"}, Value: "http://www.w3.org/2001/XMLSchema-instance"},
			{Name: xml.Name{Space: "", Local: "xmlns:xsd"}, Value: "http://www.w3.org/2001/XMLSchema"},
			{Name: xml.Name{Space: "", Local: "xmlns:soap"}, Value: "http://schemas.xmlsoap.org/soap/envelope/"},
		}
	} else {
		e.Attr = make([]xml.Attr, 0)
		for local, value := range customEnvelopeAttrs {
			e.Attr = append(e.Attr, xml.Attr{
				Name:  xml.Name{Space: "", Local: local},
				Value: value,
			})
		}
	}

	tokens.data = append(tokens.data, e)
}

func (tokens *tokenData) endEnvelope() {
	e := xml.EndElement{
		Name: xml.Name{
			Space: "",
			Local: fmt.Sprintf("%s:Envelope", soapPrefix),
		},
	}

	tokens.data = append(tokens.data, e)
}

func (tokens *tokenData) startHeader(m, n string) {
	h := xml.StartElement{
		Name: xml.Name{
			Space: "",
			Local: fmt.Sprintf("%s:Header", soapPrefix),
		},
	}

	if m == "" || n == "" {
		tokens.data = append(tokens.data, h)
		return
	}

	r := xml.StartElement{
		Name: xml.Name{
			Space: "",
			Local: m,
		},
		Attr: []xml.Attr{
			{Name: xml.Name{Space: "", Local: "xmlns"}, Value: n},
		},
	}

	tokens.data = append(tokens.data, h, r)

	return
}

func (tokens *tokenData) endHeader(m string) {
	h := xml.EndElement{
		Name: xml.Name{
			Space: "",
			Local: fmt.Sprintf("%s:Header", soapPrefix),
		},
	}

	if m == "" {
		tokens.data = append(tokens.data, h)
		return
	}

	r := xml.EndElement{
		Name: xml.Name{
			Space: "",
			Local: m,
		},
	}

	tokens.data = append(tokens.data, r, h)
}

func (tokens *tokenData) startBody(m, n string) error {
	b := xml.StartElement{
		Name: xml.Name{
			Space: "",
			Local: fmt.Sprintf("%s:Body", soapPrefix),
		},
	}

	if m == "" || n == "" {
		return fmt.Errorf("method or namespace is empty")
	}

	r := xml.StartElement{
		Name: xml.Name{
			Space: "",
			Local: m,
		},
		Attr: []xml.Attr{
			{Name: xml.Name{Space: "", Local: "xmlns"}, Value: n},
		},
	}

	tokens.data = append(tokens.data, b, r)

	return nil
}

// endToken close body of the envelope
func (tokens *tokenData) endBody(m string) {
	b := xml.EndElement{
		Name: xml.Name{
			Space: "",
			Local: fmt.Sprintf("%s:Body", soapPrefix),
		},
	}

	r := xml.EndElement{
		Name: xml.Name{
			Space: "",
			Local: m,
		},
	}

	tokens.data = append(tokens.data, r, b)
}
