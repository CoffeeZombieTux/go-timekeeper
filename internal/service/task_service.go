package service

import (
	"context"
	"fmt"
	"go-timekeeper/internal/apperror"
	"go-timekeeper/internal/model"
	apimodel "go-timekeeper/internal/model/api"
	"go-timekeeper/internal/repository"
	"go-timekeeper/internal/uow"

	"github.com/google/uuid"
)

// TaskServiceInterface represents a task service.
type TaskServiceInterface interface {
	Create(ctx context.Context, req *apimodel.CreateTaskRequest) (*model.Task, error)
	Update(ctx context.Context, req *apimodel.UpdateTaskRequest) (*model.Task, error)
	Get(ctx context.Context, id uuid.UUID, unit *uow.UnitOfWork) (*model.Task, error)
	GetByProject(
		ctx context.Context,
		projectID uuid.UUID,
		isActive *bool,
		requestedLimit,
		requestedOffset int,
	) ([]*model.Task, *apimodel.PaginationResponse, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Start(ctx context.Context, id uuid.UUID, timezone string) error
	Stop(ctx context.Context, id uuid.UUID) error
	Close(ctx context.Context, id uuid.UUID) error
}

// TaskService is a struct that implements the TaskServiceInterface.
type TaskService struct {
	taskRepo          repository.TaskRepositoryInterface
	timeRecordRepo    repository.TimeRecordRepositoryInterface
	timeRecordService TimeRecordServiceInterface
	uowManager        *uow.UnitOfWorkManager
}

// NewTaskService creates a new TaskService instance.
func NewTaskService(
	taskRepo repository.TaskRepositoryInterface,
	timeRecordRepo repository.TimeRecordRepositoryInterface,
	timeRecordService TimeRecordServiceInterface,
	uow *uow.UnitOfWorkManager,
) *TaskService {
	return &TaskService{
		taskRepo:          taskRepo,
		timeRecordRepo:    timeRecordRepo,
		timeRecordService: timeRecordService,
		uowManager:        uow,
	}
}

// Create manages a new task creation process.
func (taskService *TaskService) Create(ctx context.Context, req *apimodel.CreateTaskRequest) (*model.Task, error) {
	userId, err := getUserIdFromRequest(ctx)
	if err != nil {
		return nil, err
	}
	var task = &model.Task{
		UserID:    userId,
		ProjectID: req.ProjectID,
		Name:      req.Name,
		Status:    model.DefaultStatus,
	}

	return taskService.taskRepo.Save(ctx, task, nil)
}

// Update manages a task update process.
func (taskService *TaskService) Update(ctx context.Context, req *apimodel.UpdateTaskRequest) (*model.Task, error) {
	var task *model.Task
	err := uow.WithUnitOfWork(ctx, taskService.uowManager, func(unit *uow.UnitOfWork) error {
		var err error
		var isProjectChanged = false
		task, err = taskService.Get(ctx, req.ID, unit)
		if err != nil {
			return err
		}
		if req.ProjectID != task.ProjectID {
			isProjectChanged = true
		}
		task.Name = req.Name
		task.ProjectID = req.ProjectID

		task, err = taskService.taskRepo.Save(ctx, task, unit)
		if err != nil {
			return err
		}
		if isProjectChanged {
			timeRecords, err := taskService.timeRecordRepo.GetListByTaskForUpdate(ctx, task.ID, unit.GetTransaction())
			if err != nil {
				return err
			}
			for _, timeRecord := range timeRecords {
				timeRecord.ProjectID = req.ProjectID
				_, err = taskService.timeRecordRepo.Update(ctx, timeRecord, unit.GetTransaction())
				if err != nil {
					return err
				}
			}
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}

// Get manages a task get process.
func (taskService *TaskService) Get(ctx context.Context, id uuid.UUID, unit *uow.UnitOfWork) (*model.Task, error) {
	userId, err := getUserIdFromRequest(ctx)
	if err != nil {
		return nil, err
	}
	task, err := taskService.taskRepo.Get(ctx, id, unit)
	if err != nil {
		return nil, err
	}
	if err := checkTaskUserAccess(userId, *task); err != nil {
		return nil, err
	}
	return task, nil
}

// GetByProject manages a task get by a project process.
func (taskService *TaskService) GetByProject(
	ctx context.Context,
	projectID uuid.UUID,
	isActive *bool,
	requestedLimit,
	requestedOffset int,
) ([]*model.Task, *apimodel.PaginationResponse, error) {
	userId, err := getUserIdFromRequest(ctx)
	if err != nil {
		return nil, nil, err
	}
	params := apimodel.NewPaginationParams(requestedLimit, requestedOffset)

	totalCount, err := taskService.taskRepo.CountByProjectAndUserId(ctx, projectID, userId, isActive)
	if err != nil {
		return nil, nil, err
	}
	tasks, err := taskService.taskRepo.GetByProjectAndUserId(
		ctx,
		projectID,
		userId,
		isActive,
		params.Limit,
		params.Offset,
	)
	if err != nil {
		return nil, nil, err
	}

	currentPage := params.Offset/params.Limit + 1
	totalPages := (totalCount + params.Limit - 1) / params.Limit

	pagination := &apimodel.PaginationResponse{
		Limit:       params.Limit,
		Offset:      params.Offset,
		TotalItems:  totalCount,
		CurrentPage: currentPage,
		TotalPages:  totalPages,
	}

	if totalCount == 0 {
		pagination.CurrentPage = 0
		return []*model.Task{}, pagination, nil
	}

	if currentPage > totalPages {
		return nil, nil, apperror.New(
			apperror.CodeDBNoRowsCode,
			apperror.CodeDBNoRowsMessage,
			fmt.Sprintf("requested page %d is out of range", currentPage),
		)
	}

	if len(tasks) == 0 {
		return []*model.Task{}, pagination, nil
	}
	return tasks, pagination, nil
}

// Delete manages a task delete process.
func (taskService *TaskService) Delete(ctx context.Context, id uuid.UUID) error {
	err := uow.WithUnitOfWork(ctx, taskService.uowManager, func(unit *uow.UnitOfWork) error {
		task, err := taskService.Get(ctx, id, unit)
		if err != nil {
			return err
		}
		return taskService.taskRepo.Delete(ctx, task, unit)
	})
	if err != nil {
		return err
	}
	return nil
}

// Start manages a task start process.
func (taskService *TaskService) Start(ctx context.Context, id uuid.UUID, timezone string) error {
	err := uow.WithUnitOfWork(ctx, taskService.uowManager, func(unit *uow.UnitOfWork) error {
		const requiredStatus = model.StatusWorkingOn
		task, err := taskService.Get(ctx, id, unit)
		if err != nil {
			return err
		}
		if !checkStatusValidator(task.Status, requiredStatus) {
			return apperror.New(apperror.CodeValidationErrorCode, apperror.CodeValidationErrorMessage,
				fmt.Sprintf("required status %s is not valid for task %s", requiredStatus, task.ID),
			)
		}

		task.Status = requiredStatus
		_, err = taskService.taskRepo.Save(ctx, task, unit)
		if err != nil {
			return err
		}
		err = taskService.timeRecordService.StartTask(ctx, task, timezone, unit)
		return err
	})
	if err != nil {
		return err
	}
	return nil
}

// Stop manages a task stop process.
func (taskService *TaskService) Stop(ctx context.Context, id uuid.UUID) error {
	err := uow.WithUnitOfWork(ctx, taskService.uowManager, func(unit *uow.UnitOfWork) error {
		const requiredStatus = model.StatusCreated

		task, err := taskService.Get(ctx, id, unit)
		if err != nil {
			return err
		}
		if !checkStatusValidator(task.Status, requiredStatus) {
			return apperror.New(apperror.CodeValidationErrorCode, apperror.CodeValidationErrorMessage,
				fmt.Sprintf("required status %s is not valid for task %s", requiredStatus, task.ID),
			)
		}
		task.Status = requiredStatus
		_, err = taskService.taskRepo.Save(ctx, task, unit)
		if err != nil {
			return err
		}
		err = taskService.timeRecordService.StopTask(ctx, task, unit)
		return err
	})
	if err != nil {
		return err
	}
	return nil
}

// Close manages a task close process.
func (taskService *TaskService) Close(ctx context.Context, id uuid.UUID) error {
	err := uow.WithUnitOfWork(ctx, taskService.uowManager, func(unit *uow.UnitOfWork) error {

		const requiredStatus = model.StatusClosed
		task, err := taskService.Get(ctx, id, unit)
		if err != nil {
			return err
		}
		if !checkStatusValidator(task.Status, requiredStatus) {
			return apperror.New(apperror.CodeValidationErrorCode, apperror.CodeValidationErrorMessage,
				fmt.Sprintf("required status %s is not valid for task %s", requiredStatus, task.ID),
			)
		}
		task.Status = requiredStatus
		_, err = taskService.taskRepo.Save(ctx, task, unit)
		return err
	})
	return err
}

// checkUserAccess checks if the user is allowed to access the task.
func checkTaskUserAccess(userId uuid.UUID, task model.Task) error {
	if userId == task.UserID {
		return nil
	}
	return apperror.New(
		apperror.CodeUnauthorizedCode,
		apperror.CodeUnauthorizedMessage,
		"User not authenticated",
	)
}

// checkStatusValidator checks if the new status is valid for the old status.
func checkStatusValidator(oldStatus, newStatus model.TaskStatus) bool {
	switch newStatus {
	case model.StatusCreated:
		if oldStatus == model.StatusWorkingOn {
			return true
		}
	case model.StatusWorkingOn:
		if oldStatus == model.StatusCreated {
			return true
		}
	case model.StatusClosed:
		if oldStatus == model.StatusCreated {
			return true
		}
	}
	return false
}
