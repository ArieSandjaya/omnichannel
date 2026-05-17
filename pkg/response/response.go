package response

import "github.com/gin-gonic/gin"

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type Responder struct{}

func NewResponder() *Responder { return &Responder{} }

func (r *Responder) Success(c *gin.Context, status int, message string, data interface{}) {
	c.JSON(status, APIResponse{Success: true, Message: message, Data: data})
}

func (r *Responder) Error(c *gin.Context, status int, message string, err error) {
	resp := APIResponse{Success: false, Message: message}
	if err != nil {
		resp.Error = err.Error()
	}
	c.JSON(status, resp)
}
