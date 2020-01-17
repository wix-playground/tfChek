package git

import "log"

//NOTE: Looks like this class is not needed at all
type MultiManager interface {
	GetManagers() []Manager
	GetManagerByPath(path string) Manager
	GetManagerByOrigin(remote string) Manager
	AddManager(manager Manager)
	DeleteManager(manager Manager)
}

type MutliRepoManager struct {
	o2m map[string]Manager
	p2m map[string]Manager
	//Note: We must maintail override order. So it is needed to fill up this array from starting from the last element
	managers []Manager
}

func (m *MutliRepoManager) GetManagers() []Manager {
	return m.managers
}

func (m *MutliRepoManager) GetManagerByPath(path string) Manager {
	if m, ok := m.p2m[path]; ok {
		return m
	} else {
		return nil
	}
}

func (m *MutliRepoManager) GetManagerByOrigin(remote string) Manager {
	if m, ok := m.o2m[remote]; ok {
		return m
	} else {
		return nil
	}
}

func (m *MutliRepoManager) AddManager(manager Manager) {
	if manager != nil {
		if _, ok := m.o2m[manager.GetRemote()]; ok {
			//Replace managers order
			for i, em := range m.managers {
				if em.GetRemote() == manager.GetRemote() {
					m.managers = append(m.managers[:i], m.managers[i+1], manager)
				}
			}
		}
		p := manager.GetPath()
		m.p2m[p] = manager
		o := manager.GetRemote()
		m.o2m[o] = manager
	}
}

func (m *MutliRepoManager) DeleteManager(manager Manager) {
	p := manager.GetPath()
	if i, ok := m.p2m[p]; ok {
		delete(m.p2m, p)
		delete(m.o2m, i.GetRemote())
		for i, em := range m.managers {
			if em.GetRemote() == manager.GetRemote() {
				m.managers = append(m.managers[:i], m.managers[i+1])
			}
		}
	} else {
		if Debug {
			log.Printf("Cannot delete manager. No such path %s or remote %s exists", manager.GetPath(), manager.GetRemote())
		}
	}
}
