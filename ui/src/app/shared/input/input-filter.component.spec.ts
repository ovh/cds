import { FilterText, InputFilterComponent } from './input-filter.component';

const nbsp = InputFilterComponent.spaceAlternative;

describe('FilterText', () => {
	it('should parse key:value filters', () => {
		const res = FilterText.parse('status:Success status:Fail ref:refs/heads/main');
		expect(res.filters).toEqual({ status: ['Success', 'Fail'], ref: ['refs/heads/main'] });
		expect(res.query).toEqual([]);
	});

	it('should parse bare tokens as free text', () => {
		const res = FilterText.parse('awesome status:Success main');
		expect(res.filters).toEqual({ status: ['Success'] });
		expect(res.query).toEqual(['awesome', 'main']);
	});

	it('should decode the space alternative in keys and values', () => {
		const res = FilterText.parse(`my${nbsp}key:my${nbsp}long${nbsp}value`);
		expect(res.filters).toEqual({ 'my key': ['my long value'] });
	});

	it('should ignore empty tokens', () => {
		expect(FilterText.parse('  awesome  ')).toEqual({ filters: {}, query: ['awesome'] });
		expect(FilterText.parse(null)).toEqual({ filters: {}, query: [] });
	});

	it('should keep empty filter values unless asked to skip them', () => {
		expect(FilterText.parse('status:').filters).toEqual({ status: [''] });
		expect(FilterText.parse('status:', { skipEmptyValues: true }).filters).toEqual({});
	});

	it('should build search params keeping every free text word', () => {
		expect(FilterText.toSearchParams('awesome main type:workflow project:KEY')).toEqual({
			type: ['workflow'],
			project: ['KEY'],
			query: 'awesome main'
		});
	});

	it('should not set a query search param without free text', () => {
		expect(FilterText.toSearchParams('type:workflow')).toEqual({ type: ['workflow'] });
	});

	it('should build query params with the free text under the query key', () => {
		expect(FilterText.toQueryParams('awesome main status:Success status:Fail ref:')).toEqual({
			status: ['Success', 'Fail'],
			query: 'awesome main'
		});
	});

	it('should collapse single valued filters in query params', () => {
		expect(FilterText.toQueryParams('status:Success')).toEqual({ status: 'Success' });
	});

	it('should rebuild the filter text from query params, free text last', () => {
		expect(FilterText.fromQueryParams({ query: 'awesome main', status: 'Success' }))
			.toEqual('status:Success awesome main');
	});

	it('should encode spaces when rebuilding the filter text', () => {
		expect(FilterText.fromQueryParams({ 'my key': 'my long value' }))
			.toEqual(`my${nbsp}key:my${nbsp}long${nbsp}value`);
	});

	it('should ignore the given query param keys', () => {
		expect(FilterText.fromQueryParams({ page: 2, sort: 'started:asc', query: 'awesome' }, ['page', 'sort']))
			.toEqual('awesome');
	});

	it('should round trip filter text in canonical order through query params', () => {
		[
			'awesome',
			'awesome main',
			'status:Success',
			'status:Success status:Fail awesome',
			`my${nbsp}key:my${nbsp}value awesome`
		].forEach(text => {
			expect(FilterText.fromQueryParams(FilterText.toQueryParams(text))).toEqual(text);
		});
	});

	it('should move the free text last when rebuilding the filter text', () => {
		expect(FilterText.fromQueryParams(FilterText.toQueryParams('awesome status:Success')))
			.toEqual('status:Success awesome');
	});
});
