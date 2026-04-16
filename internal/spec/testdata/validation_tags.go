package testdata

type CreateUserRequest struct {
	Name  string `json:"name" validate:"required,min=2,max=100"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"min=0,max=150"`
}

type UpdateProfile struct {
	Bio    string `json:"bio" binding:"max=500"`
	Avatar string `json:"avatar" binding:"omitempty,url"`
}

type OrderItem struct {
	ProductID string  `json:"product_id" validate:"required,uuid"`
	Quantity  int     `json:"quantity" validate:"required,min=1,max=100"`
	Price     float64 `json:"price" validate:"required,gt=0"`
}
