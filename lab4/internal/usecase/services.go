package usecase

import (
	"NSSaDS/lab4/internal/domain"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type EchoService struct {
	port int
}

func NewEchoService(port int) domain.Service {
	return &EchoService{port: port}
}

func (s *EchoService) Name() domain.ServiceType { return domain.EchoService }
func (s *EchoService) Port() int                { return s.port }

func (s *EchoService) HandleRequest(ctx context.Context, req *domain.Request) (*domain.Response, error) {
	return &domain.Response{
		ID:        req.ID,
		Service:   s.Name(),
		Data:      []byte(fmt.Sprintf("ECHO: %s", string(req.Data))),
		Timestamp: time.Now(),
	}, nil
}

type TimeService struct {
	port int
}

func NewTimeService(port int) domain.Service {
	return &TimeService{port: port}
}

func (s *TimeService) Name() domain.ServiceType { return domain.TimeService }
func (s *TimeService) Port() int                { return s.port }

func (s *TimeService) HandleRequest(ctx context.Context, req *domain.Request) (*domain.Response, error) {
	now := time.Now()
	response := fmt.Sprintf("Current time: %s", now.Format(time.RFC3339))

	if strings.ToUpper(req.Command) == "UNIX" {
		response = fmt.Sprintf("Unix timestamp: %d", now.Unix())
	}

	return &domain.Response{
		ID:        req.ID,
		Service:   s.Name(),
		Data:      []byte(response),
		Timestamp: time.Now(),
	}, nil
}

type CalcService struct {
	port int
}

func NewCalcService(port int) domain.Service {
	return &CalcService{port: port}
}

func (s *CalcService) Name() domain.ServiceType { return domain.CalcService }
func (s *CalcService) Port() int                { return s.port }

func (s *CalcService) HandleRequest(ctx context.Context, req *domain.Request) (*domain.Response, error) {
	parts := strings.Fields(string(req.Data))
	if len(parts) < 3 {
		return &domain.Response{
			ID:        req.ID,
			Service:   s.Name(),
			Error:     fmt.Errorf("usage: <num1> <op> <num2>"),
			Timestamp: time.Now(),
		}, nil
	}

	a, err1 := strconv.ParseFloat(parts[0], 64)
	b, err2 := strconv.ParseFloat(parts[2], 64)

	if err1 != nil || err2 != nil {
		return &domain.Response{
			ID:        req.ID,
			Service:   s.Name(),
			Error:     fmt.Errorf("invalid numbers"),
			Timestamp: time.Now(),
		}, nil
	}

	op := parts[1]
	var result float64

	switch op {
	case "+":
		result = a + b
	case "-":
		result = a - b
	case "*":
		result = a * b
	case "/":
		if b == 0 {
			return &domain.Response{
				ID:        req.ID,
				Service:   s.Name(),
				Error:     fmt.Errorf("division by zero"),
				Timestamp: time.Now(),
			}, nil
		}
		result = a / b
	default:
		return &domain.Response{
			ID:        req.ID,
			Service:   s.Name(),
			Error:     fmt.Errorf("unsupported operator: %s", op),
			Timestamp: time.Now(),
		}, nil
	}

	response := fmt.Sprintf("%.2f %s %.2f = %.2f", a, op, b, result)

	return &domain.Response{
		ID:        req.ID,
		Service:   s.Name(),
		Data:      []byte(response),
		Timestamp: time.Now(),
	}, nil
}
