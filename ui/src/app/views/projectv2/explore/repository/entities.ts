import { Entity, EntityType } from 'app/model/entity.model';

/** The order of the entity tabs of a repository, workflows first as the usual entry point. */
export const ENTITY_TYPE_ORDER: Array<EntityType> = [
    EntityType.Workflow,
    EntityType.Action,
    EntityType.WorkerModel,
    EntityType.WorkflowTemplate,
    EntityType.Job
];

export const ENTITY_TYPE_LABELS: { [type in EntityType]: string } = {
    [EntityType.Workflow]: 'Workflows',
    [EntityType.Action]: 'Actions',
    [EntityType.WorkerModel]: 'Worker models',
    [EntityType.WorkflowTemplate]: 'Templates',
    [EntityType.Job]: 'Jobs'
};

/** Groups entities by type, each group sorted by name; a type without entity has no group. */
export function groupEntities(entities: Array<Entity>): Map<EntityType, Array<Entity>> {
    const groups = new Map<EntityType, Array<Entity>>();
    (entities ?? []).forEach(entity => {
        const group = groups.get(entity.type) ?? [];
        group.push(entity);
        groups.set(entity.type, group);
    });
    groups.forEach(group => group.sort((a, b) => a.name < b.name ? -1 : a.name > b.name ? 1 : 0));
    return groups;
}
