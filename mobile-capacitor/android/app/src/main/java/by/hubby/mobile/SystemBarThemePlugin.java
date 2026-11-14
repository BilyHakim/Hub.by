package by.hubby.mobile;

import android.graphics.Color;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

@CapacitorPlugin(name = "SystemBarTheme")
public class SystemBarThemePlugin extends Plugin {
    @PluginMethod
    public void setTheme(PluginCall call) {
        String color = call.getString("color", "");
        if (!color.matches("#[0-9a-fA-F]{6}")) {
            call.reject("Expected a six-digit hex color");
            return;
        }
        int background = Color.parseColor(color);
        boolean darkIcons = call.getBoolean("darkIcons", true);
        getActivity().runOnUiThread(() -> {
            ((MainActivity) getActivity()).setSystemBarTheme(background, darkIcons);
            call.resolve();
        });
    }
}
