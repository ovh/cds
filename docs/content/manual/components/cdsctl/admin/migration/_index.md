+++
title = "migration"
+++
## cdsctl admin migration

`Manage CDS Migrations`

### Synopsis

Theses commands manage CDS Migration and DO NOT concern database migrations.
	
A CDS Migration is an internal routine. This helps manage a complex data migration with code included
in CDS Engine. It's totally transpartent to CDS Users & Administrators - but these commands can help
CDS Administrators and core CDS Developers to debug something if needed.
	

### Options

```
  -h, --help   help for migration
```

### Options inherited from parent commands

```
  -f, --file string   set configuration file
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output
```

### SEE ALSO

* [cdsctl admin](/manual/components/cdsctl/admin/)	 - `Manage CDS (admin only)`
* [cdsctl admin migration cancel](/manual/components/cdsctl/admin/migration/cancel/)	 - `Cancel a CDS migration (USE WITH CAUTION)`
* [cdsctl admin migration list](/manual/components/cdsctl/admin/migration/list/)	 - `List all CDS migrations and their states`
* [cdsctl admin migration reset](/manual/components/cdsctl/admin/migration/reset/)	 - `Reset a CDS migration, so basically it put the migration status to "TO DO" (USE WITH CAUTION)`

