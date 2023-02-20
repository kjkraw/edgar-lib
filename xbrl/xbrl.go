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

// ParseXBRL reads the XBRL file contained in "r" and returns *XBRL.
func ParseXBRL(r io.Reader) (x *XBRL, err error) {
	doc := etree.NewDocument()
	_, err = doc.ReadFrom(r)
	if err != nil {
		return
	}
	x = &XBRL{doc, doc.Root()}
	return
}

// findContext uses the cik element to determine the main context in the document
func (x *XBRL) findContext(cik int) (string, error) {
	for _, e := range x.root.FindElements(fmt.Sprintf("//dei:EntityCentralIndexKey")) {
		if e.Text() == fmt.Sprintf("%010d", cik) {
			return e.SelectAttr("contextRef").Value, nil
		}
	}
	return "", fmt.Errorf("no context found for cik %010d", cik)
}

func (x *XBRL) findElement(tag string, contextId string) *etree.Element {
	return x.root.FindElement(fmt.Sprintf("//%s[@contextRef='%s']", tag, contextId))
}

// Unpack uses struct tags to find elements in a xbrl file.
// Any fields on "target" that have an "xbrl" struct tag will be filled with a value of the correct type.
// If no value is found for a specific element, the value will be left as a zero.
// An error is returned if a value is not able to be parsed.
func (x *XBRL) Unpack(target interface{}, cik int) error {
	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("invalid target for unpacking -- must be a pointer")
	}
	rv = rv.Elem()
	t := rv.Type()

	ctxId, err := x.findContext(cik)
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
