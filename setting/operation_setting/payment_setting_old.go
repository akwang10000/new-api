/**
This is the legacy payment settings file. Add any new payment settings to
payment_setting.go instead of extending this file.
*/

package operation_setting

import (
	"github.com/QuantumNous/new-api/common"
)

var PayAddress = ""
var CustomCallbackAddress = ""
var EpayId = ""
var EpayKey = ""
var Price = 7.3
var MinTopUp = 1
var USDExchangeRate = 7.3

var PayMethods = []map[string]string{
	{
		"name":  "Alipay",
		"color": "rgba(var(--semi-blue-5), 1)",
		"type":  "alipay",
	},
	{
		"name":  "WeChat",
		"color": "rgba(var(--semi-green-5), 1)",
		"type":  "wxpay",
	},
}

func UpdatePayMethodsByJsonString(jsonString string) error {
	PayMethods = make([]map[string]string, 0)
	return common.Unmarshal([]byte(jsonString), &PayMethods)
}

func PayMethods2JsonString() string {
	jsonBytes, err := common.Marshal(PayMethods)
	if err != nil {
		return "[]"
	}
	return string(jsonBytes)
}

func ContainsPayMethod(method string) bool {
	for _, payMethod := range PayMethods {
		if payMethod["type"] == method {
			return true
		}
	}
	return false
}
