package by.hubby.mobile;

import android.content.res.Configuration;
import android.graphics.Color;
import android.os.Build;
import android.os.Bundle;
import androidx.activity.EdgeToEdge;
import androidx.core.view.WindowCompat;
import androidx.core.view.WindowInsetsControllerCompat;
import com.getcapacitor.BridgeActivity;

public class MainActivity extends BridgeActivity {
    private int systemBarColor = Color.rgb(246, 244, 239);
    private boolean darkIcons = true;

    @Override
    public void onCreate(Bundle savedInstanceState) {
        registerPlugin(SystemBarThemePlugin.class);
        super.onCreate(savedInstanceState);
        EdgeToEdge.enable(this);
        applySystemBarTheme();
    }

    void setSystemBarTheme(int color, boolean useDarkIcons) {
        systemBarColor = color;
        darkIcons = useDarkIcons;
        applySystemBarTheme();
    }

    private void applySystemBarTheme() {
        // Also paints the native safe-area padding used by older Android WebViews.
        getWindow().getDecorView().setBackgroundColor(systemBarColor);
        // New Android versions draw these transparently over the native/WebView background.
        // Older versions still require explicit window colors.
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.Q) {
            getWindow().setStatusBarColor(systemBarColor);
            getWindow().setNavigationBarColor(Build.VERSION.SDK_INT < Build.VERSION_CODES.O ? Color.rgb(48, 76, 66) : systemBarColor);
        }
        WindowInsetsControllerCompat controller = WindowCompat.getInsetsController(getWindow(), getWindow().getDecorView());
        controller.setAppearanceLightStatusBars(darkIcons);
        controller.setAppearanceLightNavigationBars(darkIcons);
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            getWindow().setStatusBarContrastEnforced(false);
            getWindow().setNavigationBarContrastEnforced(false);
        }
    }

    @Override
    public void onConfigurationChanged(Configuration newConfig) {
        super.onConfigurationChanged(newConfig);
        applySystemBarTheme();
    }

    @Override
    public void onResume() {
        super.onResume();
        applySystemBarTheme();
    }
}
