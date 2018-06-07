# Warnings computed in CDS 

## Warning list

### MissingProjectVariable

* removed from EventProjectVariableAdd

* removed from EventProjectVariableUpdate

* added from EventProjectVariableDelete

### UnusedProjectVariable

* added from EventProjectVariableAdd

* removed and added from EventProjectVariableUpdate

* removed from EventProjectVariableDelete

### MissingProjectPermissionEnvironment and MissingProjectPermissionWorkflow

* added from EventProjectPermissionDelete

* removed from EventProjectPermissionAdd

### MissingProjectKey

* removed from EventProjectKeyAdd

* added from EventProjectKeyDelete

### UnusedProjectKey

* added from EventProjectKeyAdd

* removed from EventProjectKeyDelete

### MissingProjectVCSServer

* added from EventProjectVCSServerDelete

* removed from EventProjectVCSServerAdd

### UnusedProjectVCSServer

* removed from EventProjectVCSServerDelete

* added from EventProjectVCSServerAdd

## Event rules

### sdk.EventProjectVariableAdd

Remove the existing warning "MissingProjectVariable" if exists

Check if the variable is used by Workflow/Application/Environment/Pipeline and add the warning UnusedProjectVariable if it's not used 

### sdk.EventProjectVariableUpdate

If the variable name doesn't change, nothing happen

Remove the existing warning UnusedProjectVariable ( for the previous name )

Remove the existing warning MissingProjectVariable ( for the new name )

Check if the variable is used by Workflow/Application/Environment/Pipeline and add the warning UnusedProjectVariable if it's not used 

### sdk.EventProjectVariableDelete

Remove the existing warning UnusedProjectVariable

Check if the variable is used by Workflow/Application/Environment/Pipeline and add the warning MissingProjectVariable if it's used 

### sdk.EventProjectPermissionAdd

Remove the existing warning MissingProjectPermissionEnvironment

Remove the existing warning MissingProjectPermissionWorkflow

### sdk.EventProjectPermissionDelete

Check if the permission is used by Workflow/Environment and add a MissingProjectPermissionWorkflow/MissingProjectPermissionEnvironment if it's used

### sdk.EventProjectKeyAdd

Remove the existing warning "MissingProjectKey" if exists

Check if the key is used by Application/Pipeline and add the warning UnusedProjectKey if it's not used 

### sdk.EventProjectKeyDelete

Remove the existing warning UnusedProjectKey

Check if the key is used by Application/Pipeline and add the warning MissingProjectKey if it's used 

### sdk.EventProjectVCSServerAdd

Remove the existing warning MissingProjectVCSServer

Check if one application used the VCSServer and add the warning UnusedProjectVCSServer if it's not used

### sdk.EventProjectVCSServerDelete

Remove the existing warning UnusedProjectVCSServer

Check if one application used the VCSServer and add the warning MissingProjectVCSServer if it's used