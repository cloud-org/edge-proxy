package dev

import (
	"strings"

	"code.aliyun.com/openyurt/edge-proxy/pkg/kubernetes/serializer"

	apirequest "k8s.io/apiserver/pkg/endpoints/request"
)

// define label and type
const (
	funcLabel        = "type=functional"
	filterLabel      = "type=filter"
	consistencyLabel = "type=consistency"
	resourceLabel    = "type=resourceusage"
	consistencyType  = "consistency"
	resourceType     = "resourceusage"
)

//RespContentType protobuf content-type for response header value
const RespContentType = "application/vnd.kubernetes.protobuf"

// checkLabel check request labelSelector include label or not
func checkLabel(info *apirequest.RequestInfo, selector string, label string) bool {
	if info.IsResourceRequest && info.Verb == "list" &&
		(info.Resource == "pods" || info.Resource == "configmaps") &&
		strings.Contains(selector, label) { // only for consistency
		return true
	}

	return false
}

//CreateSerializer create a serializer
func CreateSerializer(info *apirequest.RequestInfo, sm *serializer.SerializerManager) *serializer.Serializer {
	return sm.CreateSerializer(RespContentType, info.APIGroup, info.APIVersion, info.Resource)
}
