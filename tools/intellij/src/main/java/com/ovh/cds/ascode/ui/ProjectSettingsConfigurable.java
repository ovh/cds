package com.ovh.cds.ascode.ui;

import com.intellij.openapi.components.ServiceManager;
import com.intellij.openapi.options.Configurable;
import com.intellij.openapi.options.ConfigurationException;
import com.intellij.openapi.util.NlsContexts;
import com.ovh.cds.ascode.service.StateService;
import org.jetbrains.annotations.Nullable;

import javax.swing.*;

public class ProjectSettingsConfigurable implements Configurable {

    private ProjectSettingsPanel myOptionsPanel = null;

    @Override
    public @NlsContexts.ConfigurableName String getDisplayName() {
        return "My CDS Settings";
    }

    @Override
    public @Nullable JComponent createComponent() {
        myOptionsPanel = new ProjectSettingsPanel();

        StateService mystate = ServiceManager.getService(StateService.class);
        JComponent component = myOptionsPanel.getMainComponent();
        myOptionsPanel.directoryJFile.setFilePath(mystate.jsonSchemaDirectory);
        return component;
    }

    @Override
    public boolean isModified() {
        return true;
    }

    @Override
    public void apply() throws ConfigurationException {
        StateService mystate = ServiceManager.getService(StateService.class);
        mystate.jsonSchemaDirectory = this.myOptionsPanel.directoryJFile.getSelectedFilePath();
    }
}
