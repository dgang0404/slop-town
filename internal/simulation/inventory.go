// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

func (b *Bakery) openContainerInReach(actorCell CellPosition, id string) bool {
	container := b.Entities[id]
	return container.Kind == Container && container.Open && inReach(actorCell, container.Placement.Cell)
}

func (b *Bakery) transferItem(actorID string, actorCell CellPosition, action ActionKind, args ActionArgs) (spatialEffect, *actionFailure) {
	item := b.Entities[args.Target]
	taking := action == ActionTake || action == ActionDiscreetTake
	if !taking && (item.Placement.HolderID != actorID || item.Kind != Item || !item.Portable) {
		return spatialEffect{}, targetUnavailable()
	}

	var kind, message string
	switch action {
	case ActionTake, ActionDiscreetTake:
		from := item.Placement
		reachable := inReach(actorCell, from.Cell)
		if from.HolderID != "" {
			reachable = b.openContainerInReach(actorCell, from.HolderID)
		}
		if !reachable {
			return spatialEffect{}, targetUnavailable()
		}
		if item.Kind != Item || !item.Portable {
			return spatialEffect{}, &actionFailure{"not_portable", "You can't pick that up."}
		}
		item.Placement = Placement{HolderID: actorID}
		kind, message = "item_taken", "You take the "+item.Name+"."
		if action == ActionDiscreetTake {
			kind, message = "item_taken_discreetly", "You discreetly take the "+item.Name+"."
		}
	case ActionPut:
		if !b.openContainerInReach(actorCell, args.Container) {
			return spatialEffect{}, targetUnavailable()
		}
		item.Placement = Placement{HolderID: args.Container}
		kind, message = "item_put", "You put the "+item.Name+" in the "+b.Entities[args.Container].Name+"."
	case ActionDrop:
		item.Placement = Placement{Cell: actorCell}
		kind, message = "item_dropped", "You drop the "+item.Name+"."
	default:
		return spatialEffect{}, &actionFailure{"invalid_request", "Unknown item action."}
	}
	payload := EventPayload{Target: args.Target, HolderID: item.Placement.HolderID, Destination: item.Placement.Cell}
	return spatialEffect{kind: kind, payload: payload, message: message, entityID: args.Target, entity: item}, nil
}
