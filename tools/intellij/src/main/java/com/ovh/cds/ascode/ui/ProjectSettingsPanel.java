package com.ovh.cds.ascode.ui;

import com.intellij.util.ui.JBUI;

import javax.swing.*;

public class ProjectSettingsPanel {

    public JFilePicker directoryJFile;

    public JComponent getMainComponent() {
        directoryJFile = new JFilePicker("Json schema directory", "browse");
        directoryJFile.setMode(JFilePicker.MODE_OPEN);

        return JBUI.Panels.simplePanel(0, 10)
                .addToTop(directoryJFile);
    }

    public String getJsonSchemaDirectory() {
        return directoryJFile.getSelectedFilePath();
    }
}
