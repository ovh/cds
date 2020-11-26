package com.ovh.cds.ascode.service;

import com.intellij.openapi.components.*;
import com.intellij.util.xmlb.XmlSerializerUtil;
import org.jetbrains.annotations.NotNull;
import org.jetbrains.annotations.Nullable;

@Service
@State(name="cds-plugin-conf", storages = {@Storage(value = "cds-plugin.xml", roamingType = RoamingType.DISABLED)})
public final class StateService implements PersistentStateComponent<StateService> {
    public String jsonSchemaDirectory;

    @Override
    public @Nullable StateService getState() {
        return this;
    }

    @Override
    public void loadState(@NotNull StateService state) {
        XmlSerializerUtil.copyBean(state, this);
        if (state.jsonSchemaDirectory == null || state.jsonSchemaDirectory.equals("")) {
            state.jsonSchemaDirectory = System.getProperty("user.home");
        }
    }
}
