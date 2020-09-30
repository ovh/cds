package com.ovh.cds.ascode;

import com.intellij.openapi.options.Configurable;
import com.intellij.openapi.options.ConfigurationException;
import com.intellij.openapi.util.NlsContexts;
import org.jetbrains.annotations.Nullable;

import javax.swing.*;

public class ProjectSettingsConfigurable implements Configurable {

    private ProjectSettingsPanel myOptionsPanel = null;

    @Override
    public @NlsContexts.ConfigurableName String getDisplayName() {
        return "My CDDSS Settings";
    }

    @Override
    public @Nullable JComponent createComponent() {
        myOptionsPanel = new ProjectSettingsPanel();
        return myOptionsPanel.getMainComponent();
    }

    @Override
    public boolean isModified() {
        return false;
    }

    @Override
    public void apply() throws ConfigurationException {

    }
}
