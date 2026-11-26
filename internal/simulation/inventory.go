// SPDX-License-Identifier: GPL-3.0-or-later
package simulation

func (b *Bakery) openContainerInReach(actorCell CellPosition, id string) bool {
	container := b.Entities[id]
	return container.Kind == Container && container.Open && inReach(actorCell, container.Placement.Cell)
}

func (b *Bakery) transferItem(actorID string, actorCell CellPosition, p Proposal) (spatialEffect, *actionFailure) {
	item := b.Entities[p.Args.Target]
	if p.Action != ActionTake && (item.Placement.HolderID != actorID || item.Kind != Item || !item.Portable) {
		return spatialEffect{}, targetUnavailable()
	}

	var kind, message string
	switch p.Action {
	case ActionTake:
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
	case ActionPut:
		if !b.openContainerInReach(actorCell, p.Args.Container) {
			return spatialEffect{}, targetUnavailable()
		}
		item.Placement = Placement{HolderID: p.Args.Container}
		kind, message = "item_put", "You put the "+item.Name+" in the "+b.Entities[p.Args.Container].Name+"."
	case ActionDrop:
		item.Placement = Placement{Cell: actorCell}
		kind, message = "item_dropped", "You drop the "+item.Name+"."
	default:
		return spatialEffect{}, &actionFailure{"invalid_request", "Unknown item action."}
	}
	b.Entities[p.Args.Target] = item
	payload := EventPayload{Target: p.Args.Target, HolderID: item.Placement.HolderID, Destination: item.Placement.Cell}
	return spatialEffect{kind, payload, message}, nil
}
