package application_user

type ApplicationUsersRequest struct {
	Users   []uint64 `json:"users" validate:"required"`
}