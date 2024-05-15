package model

// decode data
type InscriptionBRC20ModuleDeployContent struct {
	Proto       string                 `json:"p,omitempty"`
	Operation   string                 `json:"op,omitempty"`
	BRC20Name   string                 `json:"name,omitempty"`
	BRC20Source string                 `json:"source,omitempty"`
	BRC20Init   map[string]interface{} `json:"init,omitempty"`
}

type InscriptionBRC20ModuleWithdrawContent struct {
	Proto     string `json:"p,omitempty"`
	Operation string `json:"op,omitempty"`
	Module    string `json:"module,omitempty"`
	Tick      string `json:"tick,omitempty"`
	Amount    string `json:"amt,omitempty"`
}
