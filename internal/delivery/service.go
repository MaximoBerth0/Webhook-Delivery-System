package delivery

type Service struct {
	repo DeliveryRepository
}

func NewService(repo DeliveryRepository) *Service {
	return &Service{repo: repo}
}

//
