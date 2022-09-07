package types

//ConfigMapList v1.ConfigMapList compress some fields
type ConfigMapList struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	//Metadata   Metadata `json:"metadata"`
	Items []ConfigMap `json:"items"`
}

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
