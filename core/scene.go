package core

import "github.com/hajimehoshi/ebiten/v2"

type Scene interface {
	Update(g *Game) error
	Draw(screen *ebiten.Image)
}

type sceneNode struct {
	scene      Scene
	prev, next *sceneNode
}

type SceneManager struct {
	head, tail *sceneNode
}

func (m *SceneManager) Push(s Scene) {
	n := &sceneNode{scene: s, prev: m.tail}
	if m.tail == nil {
		m.head = n
	} else {
		m.tail.next = n
	}
	m.tail = n
}

func (m *SceneManager) Pop() {
	if m.tail == nil {
		return
	}
	m.tail = m.tail.prev
	if m.tail == nil {
		m.head = nil
	} else {
		m.tail.next = nil
	}
}

func (m *SceneManager) Active() Scene {
	if m.tail == nil {
		return nil
	}
	return m.tail.scene
}
