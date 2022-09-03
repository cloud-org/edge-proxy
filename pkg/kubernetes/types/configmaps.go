package types

type ConfigMapList struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	//Metadata   Metadata `json:"metadata"`
	Items []ConfigMap `json:"items"`
}

//type Metadata struct {
//	ResourceVersion string `json:"resourceVersion"`
//}

//type ManagedFields struct {
//	Manager    string    `json:"manager"`
//	Operation  string    `json:"operation"`
//	APIVersion string    `json:"apiVersion"`
//	Time       time.Time `json:"time"`
//	FieldsType string    `json:"fieldsType"`
//	FieldsV1   FieldsV1  `json:"fieldsV1"`
//}

type Metadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	//UID               string          `json:"uid"`
	//ResourceVersion   string          `json:"resourceVersion"`
	//CreationTimestamp time.Time       `json:"creationTimestamp"`
	Labels map[string]string `json:"labels"`
	//ManagedFields     []ManagedFields `json:"managedFields"`
}

type ConfigMap struct {
	Metadata Metadata          `json:"metadata"`
	Data     map[string]string `json:"data"`
}
