// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

// ActionOption names its targets as they appear in the actor's current view.
type ActionOption struct {
	Action ActionKind
	Args   ActionArgs
}

func (c *Coordinator) AvailableActions(actorID string) ([]ActionOption, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.bakery.AvailableActions(c.scope, actorID)
}

// AvailableActions lists executable commands in stable order. The caller must
// submit its choice against the scene that exists when the actor acts.
func (b *Bakery) AvailableActions(scope Scope, actorID string) ([]ActionOption, error) {
	view, err := b.Observe(scope, actorID)
	if err != nil {
		return nil, err
	}
	var options []ActionOption
	offer := func(action ActionKind, args ActionArgs) {
		if _, failure := b.planSpatial(view, action, args, action != ActionMove); failure == nil {
			options = append(options, ActionOption{Action: action, Args: args})
		}
	}
	for _, d := range []Direction{North, East, South, West} {
		offer(ActionMove, ActionArgs{Direction: d})
	}
	for _, target := range view.Targets {
		args := ActionArgs{Target: target.Name}
		switch b.Entities[target.Ref].Kind {
		case Door, Container:
			offer(ActionOpen, args)
			offer(ActionClose, args)
		case Item:
			offer(ActionTake, args)
			offer(ActionDrop, args)
			for _, dest := range view.Targets {
				if b.Entities[dest.Ref].Kind == Container {
					offer(ActionPut, ActionArgs{Target: target.Name, Container: dest.Name})
				}
			}
		}
	}
	return options, nil
}
