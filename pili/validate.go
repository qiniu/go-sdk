package pili

import (
	"net/http"
	"reflect"
	"sync"

	"github.com/go-playground/validator/v10"
)

const (
	TagName = "validate"
)

var defaultValidator = &Validator{}

type Validator struct {
	once     sync.Once
	validate *validator.Validate
}

// Validate 参数验证
func (v *Validator) Validate(obj interface{}) error {
	if obj == nil {
		return nil
	}
	value := reflect.ValueOf(obj)
	switch value.Kind() {
	case reflect.Ptr:
		return v.Validate(value.Elem().Interface())
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			if err := v.Validate(value.Index(i).Interface()); err != nil {
				return err
			}
		}
	case reflect.Struct:
		v.lazyInit()
		if err := v.validate.Struct(obj); err != nil {
			return ErrInfo(http.StatusBadRequest, err.Error())
		}
	}

	return nil
}

// lazyInit 延迟初始化
func (v *Validator) lazyInit() {
	v.once.Do(func() {
		v.validate = validator.New()
		v.validate.SetTagName(TagName)
	})
}
