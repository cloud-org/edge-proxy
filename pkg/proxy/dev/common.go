package dev

import (
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"strings"
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

// checkLabel check request labelSelector include label or not
func checkLabel(info *apirequest.RequestInfo, selector string, label string) bool {
	if info.IsResourceRequest && info.Verb == "list" &&
		(info.Resource == "pods" || info.Resource == "configmaps") &&
		strings.Contains(selector, label) { // only for consistency
		return true
	}

	return false
}
