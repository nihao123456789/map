package errorx

// CodeError 代表自定义的业务错误实体
type CodeError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *CodeError) Error() string {
	return e.Msg
}

// NewCodeError 实例化一个自定义业务错误
func NewCodeError(code int, msg string) error {
	return &CodeError{
		Code: code,
		Msg:  msg,
	}
}

// NewDefaultError 实例化一个默认的业务错误
func NewDefaultError(msg string) error {
	return NewCodeError(400, msg)
}
