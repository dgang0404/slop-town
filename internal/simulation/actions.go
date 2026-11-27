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
	seen := make(map[ActionOption]bool)
	add := func(action ActionKind, args ActionArgs) {
		if args.Target != "" {
			args.Target = b.Entities[args.Target].Name
		}
		if args.Container != "" {
			args.Container = b.Entities[args.Container].Name
		}
		option := ActionOption{Action: action, Args: args}
		if !seen[option] {
			options = append(options, option)
			seen[option] = true
		}
	}
	offer := func(action ActionKind, args ActionArgs) {
		if _, failure := b.planSpatial(view, Proposal{Action: action, Args: args}); failure == nil {
			add(action, args)
		}
	}
	for _, d := range []Direction{North, East, South, West} {
		offer(ActionMove, ActionArgs{Direction: d})
	}
	for _, target := range view.Targets {
		args := ActionArgs{Target: target.Ref}
		offer(ActionApproach, args)
		switch b.Entities[target.Ref].Kind {
		case Actor:
			offer(ActionIntroduce, args)
			if lines := b.dialogueChoices(view, target.Ref, false); len(lines) > 0 {
				add(ActionSpeak, ActionArgs{Target: target.Ref, Speech: lines[0].Line})
			}
		case Door, Container:
			offer(ActionOpen, args)
			offer(ActionClose, args)
			if b.Entities[target.Ref].Kind == Door {
				offer(ActionKnock, args)
			}
			if b.Entities[target.Ref].ProvidesCover {
				offer(ActionHide, args)
			}
		case Item:
			offer(ActionTake, args)
			offer(ActionDiscreetTake, args)
			offer(ActionDrop, args)
			for _, dest := range view.Targets {
				if b.Entities[dest.Ref].Kind == Container {
					offer(ActionPut, ActionArgs{Target: target.Ref, Container: dest.Ref})
				}
			}
		}
	}
	offer(ActionWait, ActionArgs{})
	offer(ActionRest, ActionArgs{})
	offer(ActionListen, ActionArgs{})
	if b.Entities[actorID].HiddenBehind != "" {
		offer(ActionEmerge, ActionArgs{})
	}
	return options, nil
}
