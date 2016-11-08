package dsl

import (
	"fmt"
	"unicode"

	"github.com/goadesign/goa/design"
	"github.com/goadesign/goa/eval"
)

// Request defines the data type which lists the request parameters in its
// attributes. Transport specific DSL may provide a mapping between the
// attributes and incoming request state (e.g. which attributes are initialized
// from HTTP headers, query string values or body fields in the case of HTTP)
//
// Request may appear in a Endpoint expression.
//
// Request takes one or two arguments. The first argument is either a reference
// to a type, the name of a type or a DSL function.
// If the first argument is a type or the name of a type then an optional DSL
// may be passed as second argument that further specializes the type by
// providing additional validations (e.g. list of required attributes)
//
// Examples:
//
// Endpoint("add", func() {
//     // Define request type inline
//     Request(func() {
//         Attribute("left", Int32, "Left operand")
//         Attribute("right", Int32, "Left operand")
//         Required("left", "right")
//     })
// })
//
// Endpoint("add", func() {
//     // Define request type by reference to user type
//     Request(Operands)
// })
//
// Endpoint("divide", func() {
//     // Specify required attributes on user type
//     Request(Operands, func() {
//         Required("left", "right")
//     })
// })
//
func Request(val interface{}, dsls ...func()) {
	att := endpointTypeDSL(val, dsls...)
	if att == nil {
		return
	}
	e := eval.Current().(*design.EndpointExpr)
	sn := camelize(e.Service.Name)
	en := camelize(e.Name)
	e.Request = &design.UserTypeExpr{
		AttributeExpr: att,
		TypeName:      fmt.Sprintf("%s%sRequest", en, sn),
	}
}

func endpointTypeDSL(p interface{}, dsls ...func()) *design.AttributeExpr {
	if len(dsls) > 1 {
		eval.ReportError("too many arguments")
		return nil
	}
	e, ok := eval.Current().(*design.EndpointExpr)
	if !ok {
		eval.IncompatibleDSL()
		return nil
	}
	var att *design.AttributeExpr
	var dsl func()
	switch actual := p.(type) {
	case func():
		dsl = actual
		att = &design.AttributeExpr{
			Reference: e.Service.DefaultType,
			Type:      design.Object{},
		}
	case *design.AttributeExpr:
		att = design.DupAtt(actual)
	case *design.UserTypeExpr:
		if len(dsls) == 0 {
			e.Request = actual
			return nil
		}
		att = design.DupAtt(actual.Attribute())
	case *design.MediaTypeExpr:
		att = design.DupAtt(actual.AttributeExpr)
	case string:
		ut := design.Root.UserType(actual)
		if ut == nil {
			eval.ReportError("unknown request type %s", actual)
			return nil
		}
		att = design.DupAtt(ut.Attribute())
	case design.DataType:
		att = &design.AttributeExpr{Type: actual}
	default:
		eval.ReportError("invalid Request argument, must be a type, a media type or a DSL building a type")
		return nil
	}
	if len(dsls) == 1 {
		if dsl != nil {
			eval.ReportError("invalid arguments in Request call, must be (type), (dsl) or (type, dsl)")
		}
		dsl = dsls[0]
	}
	if dsl != nil {
		eval.Execute(dsl, att)
	}
	return att
}

func camelize(str string) string {
	runes := []rune(str)
	w, i := 0, 0
	for i+1 <= len(runes) {
		eow := false
		if i+1 == len(runes) {
			eow = true
		} else if !validIdentifier(runes[i]) {
			runes = append(runes[:i], runes[i+1:]...)
		} else if spacer(runes[i+1]) {
			eow = true
			n := 1
			for i+n+1 < len(runes) && spacer(runes[i+n+1]) {
				n++
			}
			copy(runes[i+1:], runes[i+n+1:])
			runes = runes[:len(runes)-n]
		} else if unicode.IsLower(runes[i]) && !unicode.IsLower(runes[i+1]) {
			eow = true
		}
		i++
		if !eow {
			continue
		}
		runes[w] = unicode.ToUpper(runes[w])
		w = i
	}
	return string(runes)
}

// validIdentifier returns true if the rune is a letter or number
func validIdentifier(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func spacer(c rune) bool {
	switch c {
	case '_', ' ', ':', '-':
		return true
	}
	return false
}