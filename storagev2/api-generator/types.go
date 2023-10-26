package main

import (
	"errors"
	"fmt"

	"github.com/dave/jennifer/jen"
)

type (
	ServiceName           string
	StringLikeType        string
	MultipartFormDataType string
	Authorization         string
	Idempotent            string
	EncodeType            string
	ServiceBucketType     string
)

const (
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
)

func (s ServiceName) ToServiceName() (*jen.Statement, error) {
	const PKG_NAME = "github.com/qiniu/go-sdk/v7/storagev2/region"
	switch s {
	case ServiceNameUp:
		return jen.Qual(PKG_NAME, "ServiceUp"), nil
	case ServiceNameIo:
		return jen.Qual(PKG_NAME, "ServiceIo"), nil
	case ServiceNameRs:
		return jen.Qual(PKG_NAME, "ServiceRs"), nil
	case ServiceNameRsf:
		return jen.Qual(PKG_NAME, "ServiceRsf"), nil
	case ServiceNameApi:
		return jen.Qual(PKG_NAME, "ServiceApi"), nil
	case ServiceNameBucket:
		return jen.Qual(PKG_NAME, "ServiceBucket"), nil
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

func (t *StringLikeType) AddTypeToStatement(statement *jen.Statement) (*jen.Statement, error) {
	switch t.ToStringLikeType() {
	case StringLikeTypeString:
		return statement.Clone().String(), nil
	case StringLikeTypeInteger:
		return statement.Clone().Int64(), nil
	case StringLikeTypeFloat:
		return statement.Clone().Float64(), nil
	case StringLikeTypeBoolean:
		return statement.Clone().Bool(), nil
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

func (t *MultipartFormDataType) AddTypeToStatement(statement *jen.Statement) (*jen.Statement, error) {
	switch t.ToMultipartFormDataType() {
	case MultipartFormDataTypeString:
		return statement.Clone().String(), nil
	case MultipartFormDataTypeInteger:
		return statement.Clone().Int64(), nil
	case MultipartFormDataTypeUploadToken:
		return statement.Clone().Qual("github.com/qiniu/go-sdk/v7/storagev2/uptoken", "Provider"), nil
	case MultipartFormDataTypeBinaryData:
		return statement.Clone().Qual("github.com/qiniu/go-sdk/v7/internal/io", "ReadSeekCloser"), nil
	default:
		return nil, errors.New("unknown type")
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
