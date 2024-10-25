import { FlatElementTypeCondition } from "app/model/schema.model";

export class JSONFormSchema {
	types: { [key: string]: JSONFormSchemaTypeItem };
}

export class JSONFormSchemaTypeItem {
	fields: FormItem[];
	required: string[];
	oneOf: Map<string, JSONFormSchemaOneOfItem>;
}

export class JSONFormSchemaOneOfItem {
	keyFormItem: FormItem;
	fields: FormItem[];
}

export class FormItem {
	name: string;
	type: string;
	objectType?: string;
	keyMapType?: string;
	keyMapPattern?: string;
	enum?: string[];
	formOrder: number;
	condition: FlatElementTypeCondition[];
	description: string;
	pattern: string;
	onchange: string;
	mode: string;
	prefix: string;
	code: boolean;
	textarea: boolean;
}