package domain_test

import (
	"testing"

	"github.com/ishaf/cubit/internal/domain"
)

func TestNode(t *testing.T) {
	t.Run("Given valid node parameters", func(t *testing.T) {
		id := "node-1"
		name := "node-london-1"
		ip := "192.168.1.10"
		internalPort := 8081
		workerPort := 8080
		celldVer := "0.2.0"

		t.Run("When creating a new node then it initializes in active status", func(t *testing.T) {
			node, err := domain.NewNode(id, name, ip, internalPort, workerPort, celldVer)

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if node.Status != domain.NodeStatusActive {
				t.Errorf("expected status active, got %s", node.Status)
			}
			if node.Name != name {
				t.Errorf("expected name %s, got %s", name, node.Name)
			}
		})
	})

	t.Run("Given an active node", func(t *testing.T) {
		node, err := domain.NewNode("node-1", "node-london-1", "192.168.1.10", 8081, 8080, "0.2.0")
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		t.Run("When draining the node then it transitions to draining status", func(t *testing.T) {
			drainErr := node.Drain()

			if drainErr != nil {
				t.Fatalf("expected no error, got %v", drainErr)
			}
			if node.Status != domain.NodeStatusDraining {
				t.Errorf("expected status draining, got %s", node.Status)
			}
		})
	})

	t.Run("Given a node already draining", func(t *testing.T) {
		node, _ := domain.NewNode("node-1", "node-london-1", "192.168.1.10", 8081, 8080, "0.2.0")
		_ = node.Drain()

		t.Run("When draining again then it returns an invalid state error", func(t *testing.T) {
			err := node.Drain()

			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	})

	t.Run("Given empty celldVersion", func(t *testing.T) {
		t.Run("When creating a node then it defaults to DefaultCelldVersion 0.6.2", func(t *testing.T) {
			node, err := domain.NewNode("node-default", "node-default", "10.0.0.1", 8081, 8080, "")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if node.CelldVersion != "0.6.2" {
				t.Fatalf("expected node CelldVersion to default to 0.6.2, got %s", node.CelldVersion)
			}
			if domain.DefaultCelldVersion != "0.6.2" {
				t.Fatalf("expected DefaultCelldVersion 0.6.2, got %s", domain.DefaultCelldVersion)
			}
		})
	})
}
