package by.hubby.mobile;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;

import android.content.Context;
import android.security.NetworkSecurityPolicy;
import androidx.test.ext.junit.runners.AndroidJUnit4;
import androidx.test.platform.app.InstrumentationRegistry;
import org.junit.Test;
import org.junit.runner.RunWith;

@RunWith(AndroidJUnit4.class)
public class AppConfigurationTest {
    @Test
    public void mobileAppUsesItsOwnIdentityAndRequiresHttps() {
        Context context = InstrumentationRegistry.getInstrumentation().getTargetContext();
        assertEquals("by.hubby.mobile", context.getPackageName());
        assertFalse(NetworkSecurityPolicy.getInstance().isCleartextTrafficPermitted());
    }
}
