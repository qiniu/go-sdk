package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/dave/jennifer/jen"
)

type (
	MethodName            string
	ServiceName           string
	StringLikeType        string
	MultipartFormDataType string
	OptionalType          string
	Authorization         string
	Idempotent            string
	EncodeType            string
	ServiceBucketType     string
	ServiceObjectType     string
)

const (
	MethodNameGET    MethodName = http.MethodGet
	MethodNamePOST   MethodName = http.MethodPost
	MethodNamePUT    MethodName = http.MethodPut
	MethodNamePATCH  MethodName = http.MethodPatch
	MethodNameDELETE MethodName = http.MethodDelete

	ServiceNameUp     ServiceName = "up"
	ServiceNameIo     ServiceName = "io"
	ServiceNameRs     ServiceName = "rs"
	ServiceNameRsf    ServiceName = "rsf"
	ServiceNameApi    ServiceName = "api"
	ServiceNameBucket ServiceName = "uc"

	StringLikeTypeString  StringLikeType = "string"
	StringLikeTypeInteger StringLikeType = "integer"
	StringLikeTypeFloat   StringLikeType = "float"
	StringLikeTypeBoolean StringLikeType = "boolean"

	MultipartFormDataTypeString      MultipartFormDataType = "string"
	MultipartFormDataTypeInteger     MultipartFormDataType = "integer"
	MultipartFormDataTypeUploadToken MultipartFormDataType = "upload_token"
	MultipartFormDataTypeBinaryData  MultipartFormDataType = "binary_data"

	OptionalTypeRequired  OptionalType = ""
	OptionalTypeOmitEmpty OptionalType = "omitempty"
	OptionalTypeKeepEmpty OptionalType = "keepempty"
	OptionalTypeNullable  OptionalType = "nullable"

	AuthorizationNone    Authorization = ""
	AuthorizationQbox    Authorization = "Qbox"
	AuthorizationQiniu   Authorization = "Qiniu"
	AuthorizationUpToken Authorization = "UploadToken"

	IdempotentAlways  Idempotent = "always"
	IdempotentDefault Idempotent = "default"
	IdempotentNever   Idempotent = "never"

	EncodeTypeNone                EncodeType = "none"
	EncodeTypeUrlsafeBase64       EncodeType = "url_safe_base64"
	EncodeTypeUrlsafeBase64OrNone EncodeType = "url_safe_base64_or_none"

	ServiceBucketTypeNone        = ""
	ServiceBucketTypePlainText   = "plain_text"
	ServiceBucketTypeEntry       = "entry"
	ServiceBucketTypeUploadToken = "upload_token"

	ServiceObjectTypeNone      = ""
	ServiceObjectTypePlainText = "plain_text"
	ServiceObjectTypeEntry     = "entry"
)

func (s MethodName) ToString() (string, error) {
	switch s {
	case MethodNameGET, MethodNamePOST, MethodNamePUT, MethodNameDELETE:
		return string(s), nil
	case "get":
		return string(MethodNameGET), nil
	case "post":
		return string(MethodNamePOST), nil
	case "put":
		return string(MethodNamePUT), nil
	case "patch":
		return string(MethodNamePATCH), nil
	case "delete":
		return string(MethodNameDELETE), nil
	default:
		return "", errors.New("unknown method")
	}
}

func (s ServiceName) ToServiceName() (*jen.Statement, error) {
	switch s {
	case ServiceNameUp:
		return jen.Qual(PackageNameRegion, "ServiceUp"), nil
	case ServiceNameIo:
		return jen.Qual(PackageNameRegion, "ServiceIo"), nil
	case ServiceNameRs:
		return jen.Qual(PackageNameRegion, "ServiceRs"), nil
	case ServiceNameRsf:
		return jen.Qual(PackageNameRegion, "ServiceRsf"), nil
	case ServiceNameApi:
		return jen.Qual(PackageNameRegion, "ServiceApi"), nil
	case ServiceNameBucket:
		return jen.Qual(PackageNameRegion, "ServiceBucket"), nil
	default:
		return nil, errors.New("unknown type")
	}
}

func (t *StringLikeType) ToStringLikeType() StringLikeType {
	if t == nil {
		return StringLikeTypeString
	}
	switch *t {
	case StringLikeTypeString, StringLikeTypeInteger, StringLikeTypeFloat, StringLikeTypeBoolean:
		return *t
	case "":
		return StringLikeTypeString
	default:
		panic(fmt.Sprintf("unknown StringLikeType: %s", *t))
	}
}

func (t *StringLikeType) AddTypeToStatement(statement *jen.Statement, nilable bool) (*jen.Statement, error) {
	statement = statement.Clone()
	if nilable {
		statement = statement.Op("*")
	}
	switch t.ToStringLikeType() {
	case StringLikeTypeString:
		return statement.String(), nil
	case StringLikeTypeInteger:
		return statement.Int64(), nil
	case StringLikeTypeFloat:
		return statement.Float64(), nil
	case StringLikeTypeBoolean:
		return statement.Bool(), nil
	default:
		return nil, errors.New("unknown type")
	}
}

func (t *StringLikeType) GenerateConvertCodeToString(id *jen.Statement) (*jen.Statement, error) {
	switch t.ToStringLikeType() {
	case StringLikeTypeString:
		return id.Clone(), nil
	case StringLikeTypeInteger:
		return jen.Qual("strconv", "FormatInt").Call(id.Clone(), jen.Lit(10)), nil
	case StringLikeTypeFloat:
		return jen.Qual("strconv", "FormatFloat").Call(id.Clone(), jen.LitByte('g'), jen.Lit(-1), jen.Lit(64)), nil
	case StringLikeTypeBoolean:
		return jen.Qual("strconv", "FormatBool").Call(id.Clone()), nil
	default:
		return nil, errors.New("unknown type")
	}
}

func (t *StringLikeType) ZeroValue() (interface{}, error) {
	switch t.ToStringLikeType() {
	case StringLikeTypeString:
		return "", nil
	case StringLikeTypeInteger:
		return 0, nil
	case StringLikeTypeFloat:
		return 0.0, nil
	case StringLikeTypeBoolean:
		return false, nil
	default:
		return nil, errors.New("unknown type")
	}
}

func (t *MultipartFormDataType) ToMultipartFormDataType() MultipartFormDataType {
	if t == nil {
		return MultipartFormDataTypeString
	}
	switch *t {
	case MultipartFormDataTypeString, MultipartFormDataTypeInteger, MultipartFormDataTypeUploadToken, MultipartFormDataTypeBinaryData:
		return *t
	case "":
		return MultipartFormDataTypeString
	default:
		panic(fmt.Sprintf("unknown StringLikeType: %s", *t))
	}
}

func (t *MultipartFormDataType) ZeroValue() (interface{}, error) {
	switch t.ToMultipartFormDataType() {
	case MultipartFormDataTypeString:
		return "", nil
	case MultipartFormDataTypeInteger:
		return 0, nil
	case MultipartFormDataTypeUploadToken:
		return nil, nil
	case MultipartFormDataTypeBinaryData:
		return nil, nil
	default:
		return nil, errors.New("unknown type")
	}
}

func (t *MultipartFormDataType) AddTypeToStatement(statement *jen.Statement, nilable bool) (*jen.Statement, error) {
	statement = statement.Clone()
	switch t.ToMultipartFormDataType() {
	case MultipartFormDataTypeString:
		if nilable {
			statement = statement.Op("*")
		}
		return statement.String(), nil
	case MultipartFormDataTypeInteger:
		if nilable {
			statement = statement.Op("*")
		}
		return statement.Int64(), nil
	case MultipartFormDataTypeUploadToken:
		return statement.Qual(PackageNameUpToken, "Provider"), nil
	case MultipartFormDataTypeBinaryData:
		if nilable {
			statement = statement.Op("*")
		}
		return statement.Qual(PackageNameHTTPClient, "MultipartFormBinaryData"), nil
	default:
		return nil, errors.New("unknown type")
	}
}

func (t *OptionalType) ToOptionalType() OptionalType {
	if t == nil {
		return OptionalTypeRequired
	}
	switch *t {
	case OptionalTypeRequired, OptionalTypeOmitEmpty, OptionalTypeKeepEmpty, OptionalTypeNullable:
		return *t
	default:
		panic(fmt.Sprintf("unknown OptionalType: %s", *t))
	}
}

func (t *Authorization) ToAuthorization() Authorization {
	if t == nil {
		return AuthorizationNone
	}
	switch *t {
	case AuthorizationNone, AuthorizationQbox, AuthorizationQiniu, AuthorizationUpToken:
		return *t
	case "qbox":
		return AuthorizationQbox
	case "qiniu":
		return AuthorizationQiniu
	case "upload_token":
		return AuthorizationUpToken
	default:
		panic(fmt.Sprintf("unknown Authorization: %s", *t))
	}
}

func (t *Idempotent) ToIdempotent() Idempotent {
	if t == nil {
		return IdempotentDefault
	}
	switch *t {
	case IdempotentAlways, IdempotentDefault, IdempotentNever:
		return *t
	case "":
		return IdempotentDefault
	default:
		panic(fmt.Sprintf("unknown Idempotent: %s", *t))
	}
}

func (t *EncodeType) ToEncodeType() EncodeType {
	if t == nil {
		return EncodeTypeNone
	}
	switch *t {
	case EncodeTypeNone, EncodeTypeUrlsafeBase64, EncodeTypeUrlsafeBase64OrNone:
		return *t
	case "":
		return EncodeTypeNone
	default:
		panic(fmt.Sprintf("unknown EncodeType: %s", *t))
	}
}

func (t *ServiceBucketType) ToServiceBucketType() ServiceBucketType {
	if t == nil {
		return ServiceBucketTypeNone
	}
	switch *t {
	case ServiceBucketTypeNone, ServiceBucketTypePlainText, ServiceBucketTypeEntry, ServiceBucketTypeUploadToken:
		return *t
	default:
		panic(fmt.Sprintf("unknown ServiceBucketType: %s", *t))
	}
}

func (t *ServiceObjectType) ToServiceObjectType() ServiceObjectType {
	if t == nil {
		return ServiceObjectTypeNone
	}
	switch *t {
	case ServiceObjectTypeNone, ServiceObjectTypePlainText, ServiceObjectTypeEntry:
		return *t
	default:
		panic(fmt.Sprintf("unknown ServiceObjectType: %s", *t))
	}
}

func (authorization Authorization) addGetBucketNameFunc(group *jen.Group, structName string) (bool, error) {
	if authorization.ToAuthorization() == AuthorizationUpToken {
		group.Add(jen.Func().
			Params(jen.Id("request").Op("*").Id(structName)).
			Id("getBucketName").
			Params(jen.Id("ctx").Qual("context", "Context")).
			Params(jen.String(), jen.Error()).
			BlockFunc(func(group *jen.Group) {
				group.Add(jen.If(jen.Id("request").Dot("UpToken").Op("!=").Nil()).BlockFunc(func(group *jen.Group) {
					group.If(
						jen.List(jen.Id("putPolicy"), jen.Err()).Op(":=").Id("request").Dot("UpToken").Dot("GetPutPolicy").Call(jen.Id("ctx")),
						jen.Err().Op("!=").Nil(),
					).BlockFunc(func(group *jen.Group) {
						group.Return(jen.Lit(""), jen.Err())
					}).Else().BlockFunc(func(group *jen.Group) {
						group.Return(jen.Id("putPolicy").Dot("GetBucketName").Call())
					})
				}))
				group.Add(jen.Return(jen.Lit(""), jen.Nil()))
			}))
		return true, nil
	}
	return false, nil
}

func (authorization Authorization) addGetObjectNameFunc(group *jen.Group, structName string) (bool, error) {
	return false, nil
}
