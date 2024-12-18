package class

import (
	"uni-schedule-backend/internal/apperror"
	"uni-schedule-backend/internal/domain"
	"uni-schedule-backend/internal/repository"
)

type ClassService struct {
	repo         repository.ClassRepository
	scheduleRepo repository.ScheduleRepository
}

func NewClassService(repo repository.ClassRepository, scheduleRepo repository.ScheduleRepository) *ClassService {
	return &ClassService{repo: repo, scheduleRepo: scheduleRepo}
}

func (s *ClassService) Create(class domain.CreateClassDTO) (uint64, error) {
	createdClassID, err := s.repo.CreateOrSplit(class)
	if err != nil {
		return 0, err
	}

	return createdClassID, nil
}

func (s *ClassService) GetByID(id uint64) (domain.Class, error) {
	return s.repo.GetByID(id)
}

func (s *ClassService) GetAll(scheduleID uint64) ([]domain.ClassView, error) {
	classes, _, err := s.repo.GetAllViews(scheduleID)
	if err != nil {
		return classes, err
	}

	return classes, nil
}

func (s *ClassService) Update(userID uint64, id uint64, update domain.UpdateClassDTO) error {
	class, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	schedule, err := s.scheduleRepo.GetByID(class.ScheduleID)
	if err != nil {
		return err
	}
	if schedule.UserID != userID {
		return apperror.ErrDontHavePermission
	}

	return s.repo.UpdateOrSwitch(id, class.ScheduleID, update)
}

func (s *ClassService) Delete(userID uint64, id uint64) error {
	if err := s.isScheduleOwner(userID, id); err != nil {
		return err
	}
	return s.repo.Delete(id)
}

func (s *ClassService) isScheduleOwner(userID uint64, classID uint64) error {
	entry, err := s.repo.GetByID(classID)
	if err != nil {
		return err
	}
	schedule, err := s.scheduleRepo.GetByID(entry.ScheduleID)
	if err != nil {
		return err
	}
	if schedule.UserID != userID {
		return apperror.ErrDontHavePermission
	}

	return nil
}
