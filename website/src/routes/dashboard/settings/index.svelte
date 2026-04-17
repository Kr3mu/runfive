<script lang="ts">
    import { navigate } from "sv-router/generated";

    // /dashboard/settings has no content of its own — always land on Profile.
    void navigate("/dashboard/settings/profile" as any, { replace: true, params: {} });
</script>
