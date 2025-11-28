package models

type GenerateOutlineRequest struct {
	DebugPort   int      `json:"debugPort,omitempty" example:"0"`
	Website     []string `json:"website,omitempty" example:"[\"string\"]"`
	YouTube     []string `json:"youtube,omitempty" example:"[\"string\"]"`
	TextContent string   `json:"textContent,omitempty" example:"string"`
}

type GenerateOutlineResponse struct {
	Success bool        `json:"success" example:"true"`
	Message string      `json:"message,omitempty" example:"Outline generated and uploaded successfully"`
	Data    interface{} `json:"data,omitempty"`
}
