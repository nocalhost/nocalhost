package application_user

type ApplicationUsersRequest struct {
	Users   []int `json:"users" validate:"required"`
}