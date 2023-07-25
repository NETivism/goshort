package goshort

type GoShort struct {
	Short    string `json:"short"`
	Redirect string `json:"redirect"`
	Count    int    `json:"count"`
}

type GoShortResp struct {
	Success int       `json:"success"`
	Message string    `json:"message"`
	Result  []GoShort `json:"result,omitempty"`
}
