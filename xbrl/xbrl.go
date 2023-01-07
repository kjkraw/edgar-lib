package xbrl

import (
	"errors"
	"fmt"
	"github.com/beevik/etree"
	"io"
	"reflect"
	"strconv"
)

type XBRL struct {
	doc  *etree.Document
	root *etree.Element
}

func ParseXBRL(r io.Reader) (x *XBRL, err error) {
	doc := etree.NewDocument()
	_, err = doc.ReadFrom(r)
	if err != nil {
		return
	}
	x = &XBRL{doc, doc.Root()}
	return
}

// findContext searches the contexts in an XBRL document for a context with the matching cik and date string.
// "date" is the date of the instant formatted as "yyyy-mm-dd".
func (x *XBRL) findContext(cik int, instant string) (id string, err error) {
	contexts := x.root.FindElements("//context")
	instantPath := etree.MustCompilePath("./period/instant")

	for _, ctx := range contexts {
		entityElem := ctx.SelectElement("entity")
		cikElem := entityElem.SelectElement("identifier")
		if cikElem == nil || cikElem.Text() != fmt.Sprintf("%010d", cik) {
			continue
		}

		if entityElem.SelectElement("segment") != nil || entityElem.SelectElement("scenario") != nil {
			continue
		}

		instantElem := ctx.FindElementPath(instantPath)
		if instantElem == nil {
			continue
		}

		if instant == instantElem.Text() {
			id = ctx.SelectAttr("id").Value
			return
		}
	}

	err = fmt.Errorf("no context found for cik \"%s\" and instant \"%s\"", cik, instant)
	return
}

func (x *XBRL) findElement(tag string, contextId string) *etree.Element {
	return x.root.FindElement(fmt.Sprintf("//%s[@contextRef='%s']", tag, contextId))
}

func (x *XBRL) Unpack(target interface{}, cik int, instant string) error {
	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("invalid target for unpacking -- must be a pointer")
	}
	rv = rv.Elem()
	t := rv.Type()

	ctxId, err := x.findContext(cik, instant)
	if err != nil {
		return err
	}

	for i := 0; i < t.NumField(); i++ {
		fVal := rv.Field(i)
		if !fVal.IsValid() {
			return errors.New("invalid field")
		}

		fType := t.Field(i)
		if xmlTag, ok := fType.Tag.Lookup("xbrl"); ok {
			elem := x.findElement(xmlTag, ctxId)
			if elem == nil {
				continue
			}

			rawVal := elem.Text()
			if len(rawVal) == 0 {
				continue
			}

			var v interface{}
			switch fVal.Interface().(type) {
			case string:
				v = rawVal
			case int, int32:
				v, err = strconv.Atoi(rawVal)
			case int64:
				v, err = strconv.ParseInt(rawVal, 10, 64)
			default:
				err = errors.New(fmt.Sprintf("no conversion for type: %s", reflect.TypeOf(fVal.Interface())))
			}
			if err != nil {
				return err
			}

			fVal.Set(reflect.ValueOf(v))
		}
	}
	return nil
}
