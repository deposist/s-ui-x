package service

import (
	"strings"

	"github.com/deposist/s-ui-x/util/common"

	"google.golang.org/protobuf/encoding/protowire"
)

// V2Ray/Xray geosite.dat schema subset:
//
//	message GeoSiteList { repeated GeoSite entry = 1; }
//	message GeoSite { string country_code = 1; repeated Domain domain = 2; }
//	message Domain { enum Type { Plain=0; Regex=1; Domain=2; Full=3; } Type type = 1; string value = 2; }
//
// Attributes are intentionally ignored; presets only need domain matchers.
const (
	v2rayDomainTypePlain  int32 = 0
	v2rayDomainTypeRegex  int32 = 1
	v2rayDomainTypeDomain int32 = 2
	v2rayDomainTypeFull   int32 = 3
)

type v2rayDomain struct {
	Type  int32
	Value string
}

func readV2RayGeositeCategory(data []byte, category string) ([]v2rayDomain, error) {
	for len(data) > 0 {
		number, wireType, value, rest, err := consumeProtoField(data)
		if err != nil {
			return nil, err
		}
		data = rest
		if number != 1 || wireType != protowire.BytesType {
			continue
		}
		code, domains, err := parseV2RayGeoSite(value)
		if err != nil {
			return nil, err
		}
		if strings.EqualFold(code, category) {
			return domains, nil
		}
	}
	return nil, common.NewError("geosite category ", category, " not found")
}

func parseV2RayGeoSite(data []byte) (string, []v2rayDomain, error) {
	var code string
	var domains []v2rayDomain
	for len(data) > 0 {
		number, wireType, value, rest, err := consumeProtoField(data)
		if err != nil {
			return "", nil, err
		}
		data = rest
		switch number {
		case 1:
			if wireType == protowire.BytesType {
				code = string(value)
			}
		case 2:
			if wireType != protowire.BytesType {
				continue
			}
			domain, err := parseV2RayDomain(value)
			if err != nil {
				return "", nil, err
			}
			domains = append(domains, domain)
		}
	}
	return code, domains, nil
}

func parseV2RayDomain(data []byte) (v2rayDomain, error) {
	domain := v2rayDomain{Type: v2rayDomainTypePlain}
	for len(data) > 0 {
		number, wireType, value, rest, err := consumeProtoField(data)
		if err != nil {
			return v2rayDomain{}, err
		}
		data = rest
		switch number {
		case 1:
			if wireType != protowire.VarintType {
				continue
			}
			typ, n := protowire.ConsumeVarint(value)
			if n < 0 {
				return v2rayDomain{}, protowire.ParseError(n)
			}
			switch typ {
			case 0:
				domain.Type = v2rayDomainTypePlain
			case 1:
				domain.Type = v2rayDomainTypeRegex
			case 2:
				domain.Type = v2rayDomainTypeDomain
			case 3:
				domain.Type = v2rayDomainTypeFull
			default:
				return v2rayDomain{}, common.NewError("unsupported geosite domain type: ", typ)
			}
		case 2:
			if wireType == protowire.BytesType {
				domain.Value = string(value)
			}
		}
	}
	return domain, nil
}

func consumeProtoField(data []byte) (protowire.Number, protowire.Type, []byte, []byte, error) {
	number, wireType, tagLen := protowire.ConsumeTag(data)
	if tagLen < 0 {
		return 0, 0, nil, nil, protowire.ParseError(tagLen)
	}
	valueData := data[tagLen:]
	valueLen := protowire.ConsumeFieldValue(number, wireType, valueData)
	if valueLen < 0 {
		return 0, 0, nil, nil, protowire.ParseError(valueLen)
	}
	value := valueData[:valueLen]
	if wireType == protowire.BytesType {
		bytesValue, n := protowire.ConsumeBytes(valueData)
		if n < 0 {
			return 0, 0, nil, nil, protowire.ParseError(n)
		}
		value = bytesValue
	}
	return number, wireType, value, valueData[valueLen:], nil
}
