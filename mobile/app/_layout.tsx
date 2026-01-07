import { Stack } from 'expo-router';
import { PaperProvider } from 'react-native-paper';
import { AppProvider } from '@/contexts/AppProvider';
import { theme } from '@/theme';

export default function RootLayout() {
  return (
    <PaperProvider theme={theme}>
      <AppProvider>
        <Stack
          screenOptions={{
            headerStyle: {
              backgroundColor: theme.colors.primary,
            },
            headerTintColor: theme.colors.onPrimary,
            headerTitleStyle: {
              fontWeight: 'bold',
            },
          }}
        >
          <Stack.Screen
            name="index"
            options={{
              title: 'Devices',
            }}
          />
          <Stack.Screen
            name="device/[id]"
            options={{
              title: 'Device Details',
            }}
          />
          <Stack.Screen
            name="device/authenticate"
            options={{
              title: 'Authenticate',
              presentation: 'modal',
            }}
          />
        </Stack>
      </AppProvider>
    </PaperProvider>
  );
}
