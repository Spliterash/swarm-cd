package swarmcd

import (
	"fmt"
	"sync"
	"time"
)

var stackStatus map[string]*StackStatus = map[string]*StackStatus{}
var stacks []*swarmStack

func Run() {
	logger.Info("starting SwarmCD")
	for {
		var waitGroup sync.WaitGroup

		logger.Debug("pulling changes...")
		for _, repo := range repos {
			r, err := repo.pullChanges("master")
			if err != nil {
				logger.Error(err.Error())
				return
			}
			repo.revision = r
			logger.Debug("changes pulled", "revision", r)
		}

		logger.Info("updating stacks...")
		for _, swarmStack := range stacks {
			waitGroup.Add(1)
			go updateStackThread("init-"+swarmStack.repo.revision, swarmStack, &waitGroup)
		}
		waitGroup.Wait()
		logger.Info("waiting for the update interval")
		time.Sleep(time.Duration(Config.UpdateInterval) * time.Second)
	}
}

func updateStackThread(revision string, swarmStack *swarmStack, waitGroup *sync.WaitGroup) {
	repoLock := swarmStack.repo.lock
	repoLock.Lock()
	defer repoLock.Unlock()
	defer waitGroup.Done()

	logger.Info(fmt.Sprintf("updating %s stack", swarmStack.name))
	err := swarmStack.updateStack()
	if err != nil {
		stackStatus[swarmStack.name].Error = err.Error()
		logger.Error(err.Error())
		return
	}

	stackStatus[swarmStack.name].Error = ""
	stackStatus[swarmStack.name].Revision = revision
	logger.Info(fmt.Sprintf("done updating %s stack", swarmStack.name))
}

func UpdateAllStackInRepo(repoName string) {
	logger.Info("Update webhook stacks for repo " + repoName)

	for _, repo := range repos {
		r, err := repo.pullChanges("master")
		if err != nil {
			logger.Error(err.Error())
			return
		}
		repo.revision = r
		logger.Debug("changes pulled", "revision", r)
	}

	for _, stack := range stacks {
		if stack.repo.name == repoName {
			logger.Info("Start update " + stack.name)
			stack.repo.lock.Lock()
			err := stack.updateStack()
			if err != nil {
				logger.Error(err.Error())
				stackStatus[stack.name].Error = err.Error()
				continue
			} else {
				stackStatus[stack.name].Error = ""
				stackStatus[stack.name].Revision = stack.repo.revision
			}

			stack.repo.lock.Unlock()
		}
	}
}

func GetStackStatus() map[string]*StackStatus {
	return stackStatus
}
