package network

import (
	"NSSaDS/lab4/internal/domain"
	"sync"
)

type ServiceRegistry struct {
	services map[domain.ServiceType]domain.Service
	ports    map[int]domain.ServiceType
	mutex    sync.RWMutex
}

func NewServiceRegistry() domain.ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[domain.ServiceType]domain.Service),
		ports:    make(map[int]domain.ServiceType),
	}
}

func (sr *ServiceRegistry) RegisterService(service domain.Service) error {
	sr.mutex.Lock()
	defer sr.mutex.Unlock()

	serviceType := service.Name()
	port := service.Port()

	if _, exists := sr.services[serviceType]; exists {
		return domain.ErrServiceExists
	}

	if _, exists := sr.ports[port]; exists {
		return domain.ErrPortInUse
	}

	sr.services[serviceType] = service
	sr.ports[port] = serviceType

	return nil
}

func (sr *ServiceRegistry) GetService(serviceType domain.ServiceType) (domain.Service, error) {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	service, exists := sr.services[serviceType]
	if !exists {
		return nil, domain.ErrServiceNotFound
	}

	return service, nil
}

func (sr *ServiceRegistry) ListServices() []domain.ServiceType {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	services := make([]domain.ServiceType, 0, len(sr.services))
	for serviceType := range sr.services {
		services = append(services, serviceType)
	}

	return services
}

func (sr *ServiceRegistry) GetServicePort(serviceType domain.ServiceType) (int, error) {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	service, exists := sr.services[serviceType]
	if !exists {
		return 0, domain.ErrServiceNotFound
	}

	return service.Port(), nil
}

func (sr *ServiceRegistry) GetServiceByPort(port int) (domain.Service, error) {
	sr.mutex.RLock()
	defer sr.mutex.RUnlock()

	serviceType, exists := sr.ports[port]
	if !exists {
		return nil, domain.ErrServiceNotFound
	}

	return sr.services[serviceType], nil
}
