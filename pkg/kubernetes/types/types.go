package types

type ResourceCreateReq struct {
	Kind       string   `json:"kind"`
	APIVersion string   `json:"apiVersion"`
	Metadata   Metadata `json:"metadata"`
	Data       Data     `json:"data"`
}

type Labels struct {
	RaceID string `json:"race_id"`
	TaskID string `json:"task_id"`
	Type   string `json:"type"` // consistency: 一致性，resourceusage: 资源占用
}

type Metadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	//CreationTimestamp interface{} `json:"creationTimestamp"`
	Labels Labels `json:"labels"`
}

type Data struct {
	Test string `json:"test"`
}
