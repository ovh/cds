package com.ovh.cds.ascode;

import com.intellij.openapi.ui.LabeledComponent;
import com.intellij.util.ui.JBUI;

import javax.swing.*;
import java.awt.*;

public class ProjectSettingsPanel {

    public JComponent getMainComponent() {

        final LabeledComponent<JComboBox> component = new LabeledComponent<>();
        component.setText("BLABLA");
        component.setLabelLocation(BorderLayout.WEST);

        /*
        component.setComponent(myProfilesComboBox);
        ElementProducer<ScopeSetting> producer = new ElementProducer<ScopeSetting>() {
            @Override
            public ScopeSetting createElement() {
                return new ScopeSetting(CustomScopesProviderEx.getAllScope(), myProfilesModel.getAllProfiles().values().iterator().next());
            }

            @Override
            public boolean canCreateElement() {
                return true;
            }
        };
         */
        //ToolbarDecorator decorator = ToolbarDecorator.createDecorator(myScopeMappingTable, producer)
        //        .setAddActionUpdater(e -> !myProfilesModel.getAllProfiles().isEmpty());
        return JBUI.Panels.simplePanel(0, 10)
                .addToTop(component);
                //.addToCenter(decorator.createPanel())
                //.addToBottom(myScopesLink);
    }
}
