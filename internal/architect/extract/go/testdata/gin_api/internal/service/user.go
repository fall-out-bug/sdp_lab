package service

type UserService struct {
	// Dependencies
}

func (s *UserService) GetUser(id string) (*User, error) {
	return &User{ID: id, Name: "Test"}, nil
}

type User struct {
	ID   string
	Name string
}
