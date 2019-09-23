package binding

import (
	"github.com/pkg/errors"
	"reflect"
	"sync"

	"gopkg.in/go-playground/validator.v9"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	zh_translations "gopkg.in/go-playground/validator.v9/translations/zh"
)

type defaultValidator struct {
	once     sync.Once
	validate *validator.Validate
	trans ut.Translator
}

var _ StructValidator = &defaultValidator{}

func (v *defaultValidator) ValidateStruct(obj interface{}) error {
	if kindOfData(obj) == reflect.Struct {
		v.lazyinit()
		if err := v.validate.Struct(obj); err != nil {
			errs := err.(validator.ValidationErrors)
			errStr := "请求参数错误:"
			for _,v := range errs.Translate(v.trans){
				errStr += v+";"
			}
			return errors.New(errStr)
		}
	}
	return nil
}

func (v *defaultValidator) RegisterValidation(key string, fn validator.Func) error {
	v.lazyinit()
	return v.validate.RegisterValidation(key, fn)
}

func (v *defaultValidator) lazyinit() {
	v.once.Do(func() {

		zh := zh.New()
		uni := ut.New(zh, zh)

		v.trans, _ = uni.GetTranslator("zh")

		v.validate = validator.New()
		zh_translations.RegisterDefaultTranslations(v.validate, v.trans)
	})
}

func kindOfData(data interface{}) reflect.Kind {
	value := reflect.ValueOf(data)
	valueType := value.Kind()
	if valueType == reflect.Ptr {
		valueType = value.Elem().Kind()
	}
	return valueType
}
