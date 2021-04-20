package web_broker

type (
	GetValueReq struct {
		Key string
	}

	PutValueReq struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
)
