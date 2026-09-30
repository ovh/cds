import { Entity, EntityType } from 'app/model/entity.model';
import { groupEntities } from './entities';

const entity = (type: EntityType, name: string): Entity => <Entity>{ type, name, file_path: `.cds/${type}/${name}.yml` };

describe('groupEntities', () => {
    it('groups by type and sorts each group by name', () => {
        const groups = groupEntities([
            entity(EntityType.Workflow, 'deploy'),
            entity(EntityType.Action, 'build-image'),
            entity(EntityType.Workflow, 'checkout'),
            entity(EntityType.WorkerModel, 'debian12'),
            entity(EntityType.Workflow, 'cache')
        ]);

        expect([...groups.keys()]).toEqual([EntityType.Workflow, EntityType.Action, EntityType.WorkerModel]);
        expect(groups.get(EntityType.Workflow).map(e => e.name)).toEqual(['cache', 'checkout', 'deploy']);
        expect(groups.get(EntityType.Action).map(e => e.name)).toEqual(['build-image']);
    });

    it('leaves out the types that have no entity', () => {
        const groups = groupEntities([entity(EntityType.Action, 'setup-go')]);

        expect(groups.has(EntityType.Workflow)).toBeFalse();
        expect(groups.size).toBe(1);
    });

    it('returns an empty map for no entity at all', () => {
        expect(groupEntities([]).size).toBe(0);
        expect(groupEntities(null).size).toBe(0);
    });
});
